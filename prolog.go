package prolog

import (
	"bytes"
	"fmt"
	"strings"
)

/*
	Short functions:
	A   Atom
	And ConjGoal
	CT  *ComplexTerm
	Or  DisjGoal
	R   Rule
	V   Variable
*/

// Constants for Term Types.
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

func IsGlobalVarName(str string) bool {
	return strings.HasPrefix(str, _VAR_GLOBAL_PREFIX)
}

func V(str string) Variable {
	if IsGlobalVarName(str) {
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

func CT(functor Atom, args ...Term) *ComplexTerm {
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

// Constants for Goal Types. Returned by Goal.Type
const (
	gtConj = iota
	gtDisj
	gtComplex // *ComplexTerm
	gtMatch   // Term = Term
)

type Goal interface {
	GoalType() int
}

type ConjGoal []Goal

func And(goals ...Goal) ConjGoal {
	return ConjGoal(goals)
}

func (cg ConjGoal) String() string {
	var buf bytes.Buffer
	for i, g := range cg {
		if i > 0 {
			buf.WriteString(",\n")
		}
		buf.WriteString("    " + fmt.Sprint(g))
	}
	return buf.String()
}

func (cg ConjGoal) GoalType() int {
	return gtConj
}

type DisjGoal []Goal

func (dg DisjGoal) GoalType() int {
	return gtDisj
}

func Or(goals ...Goal) DisjGoal {
	return DisjGoal(goals)
}

func (cg *ComplexTerm) GoalType() int {
	return gtComplex
}

type MatchGoal struct {
	L, R Term
}

type Rule struct {
	Head *ComplexTerm
	Body Goal
}

func R(head *ComplexTerm, goals ...Goal) Rule {
	return Rule{Head: head, Body: ConjGoal(goals)}
}

func (r Rule) String() string {
	var buf bytes.Buffer
	buf.WriteString(r.Head.String())
	if r.Body != nil {
		buf.WriteString(" :- \n")
		buf.WriteString(fmt.Sprint(r.Body))
	}
	buf.WriteRune('.')
	return buf.String()
}

/*****************
	Machine Type
*****************/

type Machine struct {
	rules map[string][]Rule
}

func (m *Machine) AddFact(head *ComplexTerm) {
	m.AddRule(Rule{Head: head})
}

func (m *Machine) AddRule(rule Rule) {
	key := rule.Head.Key()
	m.rules[key] = append(m.rules[key], rule)
	fmt.Println(rule)
}

const S_FACT = " FACT "

type Context map[Variable]Term

func isGlobalVar(v Variable, sCtx Context) bool {
	_, ok := sCtx[v]
	return ok
}

func setVar(v Variable, vl Term, ctx, sCtx Context) {
	if isGlobalVar(v, sCtx) {
		sCtx[v] = vl
	} else {
		ctx[v] = vl
	}
}

func genGlobalVar(sCtx Context) Variable {
	v := genUniqueVar()
	sCtx[v] = nil
	return v
}

// t has been instanticated before calling
func globalize(t Term, ctx, sCtx Context) (cT Term) {
	switch t.Type() {
	case ttVar:
		v := t.(Variable)
		if isGlobalVar(v, sCtx) {
			// v has been a global variable
			return t
		}

		// otherwise, globalize it
		sV := genGlobalVar(sCtx)
		ctx[v] = sV
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
		if i := ctx[v]; i != nil {
			t = i
			continue
		}
		if i := sCtx[v]; i != nil {
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
				sV := genGlobalVar(sCtx)
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

func matchHead(ruleHead, formHead *ComplexTerm) (mRuleCtx, mFormCtx Context) {
	ruleCtx, formCtx, sCtx := make(Context), make(Context), make(Context)
	for i, ruleArg := range ruleHead.Args {
		formArg := formHead.Args[i]
		if !matchTerm(ruleArg, formArg, ruleCtx, formCtx, sCtx) {
			return nil, nil
		}
	}

	for v, vl := range ruleCtx {
		ruleCtx[v] = fullInstantiate(vl, ruleCtx, sCtx)
	}
	for v, vl := range formCtx {
		formCtx[v] = fullInstantiate(vl, formCtx, sCtx)
	}

	return ruleCtx, formCtx
}

func mulContext(b, s Context) Context {
	c := make(Context)
	for v, vl := range b {
		c[v] = vl
	}
	for v, vl := range s {
		c[v] = vl
	}

	return c
}

func (m *Machine) proveGoal(goal Goal, ctx Context, solutions chan Context) {
	switch goal.GoalType() {
	case gtConj:
		cg := goal.(ConjGoal)
		if len(cg) == 0 {
			solutions <- nil
			close(solutions)
			return
		}
		g := cg[0]
		slns := make(chan Context)
		go m.proveGoal(g, ctx, slns)
		for dctx := range slns {
			ctx2 := mulContext(ctx, dctx)
			slns2 := make(chan Context)
			go m.proveGoal(cg[1:], ctx2, slns2)
			for dctx2 := range slns2 {
				sln3 := mulContext(ctx2, dctx2)
				solutions <- sln3
			}
		}
		close(solutions)

	case gtComplex:
		ct := goal.(*ComplexTerm)
		ct = fullInstantiate(ct, ctx, nil).(*ComplexTerm)

		slns := make(chan Context)
		go m.Match(ct, slns)
		for dctx := range slns {
			//fmt.Println(indent, "G", goal, ctx, ct, dctx)
			ctx2 := mulContext(ctx, dctx)
			solutions <- ctx2
		}
		close(solutions)

	default:
		panic(fmt.Sprintf("Goal not supported: %s", goal))
	}
}

func subContext(aCtx, bCtx Context) Context {
	cCtx := mulContext(aCtx, bCtx)
	ctx := make(Context)
	for v, vl := range aCtx {
		ctx[v] = fullInstantiate(vl, cCtx, nil)
	}

	return ctx
}

var indent string

func (m *Machine) Match(form *ComplexTerm, solutions chan Context) {
	//fmt.Println(indent, "Match:", form)
	indent += "    "
	defer func() { indent = indent[:len(indent)-4] }()

	rules := m.rules[form.Key()]
	for _, rule := range rules {
		ruleCtx, formCtx := matchHead(rule.Head, form)
		if formCtx == nil {
			// head not matched
			continue
		}
		//fmt.Println(indent, "Head", form, formCtx, rule.Head, ruleCtx)

		if rule.Body == nil {
			//fmt.Println(indent, form, "Fact", rule.Head, formCtx)
			solutions <- formCtx
			continue
		}

		slns := make(chan Context)
		go m.proveGoal(rule.Body, ruleCtx, slns)
		for dctx := range slns {
			//fmt.Println(indent, "sln:", sln, mFormHead)
			sln2 := subContext(formCtx, dctx)
			solutions <- sln2
		}
	}
	close(solutions)
}

func NewMachine() *Machine {
	return &Machine{rules: map[string][]Rule{}}
}
