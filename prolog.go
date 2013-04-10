package prolog

import (
	"bytes"
	"fmt"
	"strings"
)

const (
	ttAtom = iota
	ttVar
	ttComplex
)

type Term interface {
	Type() int
}

type Atom string

func A(str string) Atom {
	return Atom(str)
}

func (at Atom) Type() int {
	return ttAtom
}

type Variable string

const _VAR_GLOBAL_PREFIX = "_AUTO_"
const _VAR_GLOBAL_FMT = "_AUTO_%d"

func IsGlobalVar(str string) bool {
	return strings.HasPrefix(str, _VAR_GLOBAL_PREFIX)
}

func V(str string) Variable {
	if IsGlobalVar(str) {
		panic(str + " is a GLOBAL VARIABLE name")
	}
	return Variable(str)
}

func _v(str string) Variable {
	return Variable(str)
}

func (v Variable) Type() int {
	return ttVar
}

var gUniqueVarChan chan Variable

func init() {
	gUniqueVarChan = make(chan Variable, 10)
	go func() {
		counter := 0
		for {
			gUniqueVarChan <- _v(fmt.Sprintf(_VAR_GLOBAL_FMT, counter))
			counter++
		}
	}()
}

func genUniqueVar() Variable {
	return <-gUniqueVarChan
}

type ComplexTerm struct {
	Functor Atom
	Args    []Term
}

func NewComplexTerm(functor Atom, args ...Term) *ComplexTerm {
	return &ComplexTerm{Functor: functor, Args: args}
}

func (ct *ComplexTerm) Type() int {
	return ttComplex
}

func (ct *ComplexTerm) Key() string {
	return fmt.Sprintf("%s/%d", ct.Functor, len(ct.Args))
}

func (ct *ComplexTerm) String() string {
	var buf bytes.Buffer
	buf.Write([]byte(ct.Functor))
	if len(ct.Args) > 0 {
		buf.WriteRune('(')
		for i, arg := range ct.Args {
			if i > 0 {
				buf.Write([]byte(", "))
			}
			buf.WriteString(fmt.Sprint(arg))
		}
		buf.WriteRune(')')
	}
	return buf.String()
}

type Rule struct {
	head *ComplexTerm
	body string
}

/*****************
	Machine Type
*****************/

type Machine struct {
	rules map[string][]Rule
}

func (m *Machine) AddFact(head *ComplexTerm) {
	key := head.Key()
	m.rules[key] = append(m.rules[key], Rule{head: head})
	fmt.Println("Adding fact:", head)
}

const S_FACT = " FACT "

type Context map[Variable]Term


func setVar(v Variable, vl Term, ctx, sCtx Context) {
	if IsGlobalVar(string(v)) {
		sCtx[v] = vl
	} else {
		ctx[v] = vl
	}
}

// t has been instanticated before calling
func globalize(t Term, ctx, sCtx Context) (cT Term) {
	switch t.Type() {
		case ttVar:
			v := t.(Variable)
			if IsGlobalVar(string(v)) {
				return t
			}
			sV := genUniqueVar()
			setVar(v, sV, ctx, sCtx)
			return sV
			
		case ttComplex:
			ct := t.(*ComplexTerm)
			gArgs := make([]Term, len(ct.Args))
			for i, arg := range ct.Args {
				arg = instantiate(arg, ctx, sCtx)
				gArgs[i] = globalize(arg, ctx, sCtx)
			}
			return NewComplexTerm(ct.Functor, gArgs...)
	}
	return t
}

func instantiate(t Term, ctx, sCtx Context) Term {
	for t.Type() == ttVar {
		v := t.(Variable)
		if i, ok := ctx[v]; ok {
			t = i
			continue
		}
		if i, ok := sCtx[v]; ok {
			t = i
			continue
		}
		
		break
	}
	return t
}

func fullInstantiate(t Term, lCtx, sCtx Context) Term {
	t = instantiate(t, lCtx, sCtx)
	switch t.Type() {
		case ttComplex:
			ct := t.(*ComplexTerm)
			newArgs := make([]Term, len(ct.Args))
			for i, arg := range ct.Args {
				newArgs[i] = fullInstantiate(arg, lCtx, sCtx)
			}
			return NewComplexTerm(ct.Functor, newArgs...)
	}
	
	return t
}

func matchTerm(L, R Term, lCtx, rCtx, sCtx Context) (succ bool) {
	if L.Type() == ttVar {
		L = instantiate(L, lCtx, sCtx)
	}
	if R.Type() == ttVar {
		R = instantiate(R, rCtx, sCtx)
	}

	// clause 1
	if L.Type() == ttAtom && R.Type() == ttAtom {
		l, r := L.(Atom), R.(Atom)
		return l == r
	}

	// clause 2
	if L.Type() == ttVar {
		lV := L.(Variable)
		if R.Type() == ttVar {
			// both Variable's
			rV := R.(Variable)
			if lV != rV {
				sV := genUniqueVar()
				setVar(lV, sV, lCtx, sCtx)
				setVar(rV, sV, rCtx, sCtx)
				// Otherwise already matche
			}
		} else {
			// L <= R
			rT := globalize(R, rCtx, sCtx)
			setVar(lV, rT, lCtx, sCtx)
		}
		return true
	}

	if R.Type() == ttVar {
		// L => R
		rV := R.(Variable)
		lT := globalize(L, lCtx, sCtx)
		setVar(rV, lT, rCtx, sCtx)
		return true
	}

	// clause 3
	if L.Type() == ttComplex && R.Type() == ttComplex {
		cL, cR := L.(*ComplexTerm), R.(*ComplexTerm)
		if cL.Functor != cR.Functor || len(cL.Args) != len(cR.Args) {
			return false
		}

		for i, lArg := range cL.Args {
			rArg := cR.Args[i]
			if !matchTerm(lArg, rArg, lCtx, rCtx, sCtx) {
				return false
			}
		}

		return true
	}

	return false
}

func matchHead(ruleHead, formHead *ComplexTerm) (mRuleHead, mFormHead *ComplexTerm) {
	ruleCtx, formCtx, sCtx := make(Context), make(Context), make(Context)
	for i, ruleArg := range ruleHead.Args {
		formArg := formHead.Args[i]
		if !matchTerm(ruleArg, formArg, ruleCtx, formCtx, sCtx) {
			return nil, nil
		}
	}

	mRuleArgs := make([]Term, len(ruleHead.Args))
	mFormArgs := make([]Term, len(formHead.Args))
	
	for i := range ruleHead.Args {
		mRuleArgs[i] = fullInstantiate(ruleHead.Args[i], ruleCtx, sCtx)
		mFormArgs[i] = fullInstantiate(formHead.Args[i], formCtx, sCtx)
	}
	fmt.Println("ruleCtx:", ruleCtx, ", formCtx:", formCtx, ", sCtx:", sCtx)
	
	return NewComplexTerm(ruleHead.Functor, mRuleArgs...), NewComplexTerm(formHead.Functor, mFormArgs...)
}

func (m *Machine) Match(head *ComplexTerm, solutions chan *ComplexTerm) {
	rules := m.rules[head.Key()]
	for _, rule := range rules {
		_, mFormHead := matchHead(rule.head, head)
		if mFormHead != nil {
			solutions <- mFormHead
		}
	}
	close(solutions)
}

func NewMachine() *Machine {
	return &Machine{rules: map[string][]Rule{}}
}
