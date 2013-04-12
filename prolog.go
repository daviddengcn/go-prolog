package prolog

import (
	"bytes"
	"fmt"
)

/*
	Short functions:
	A   Atom
	V   Variable
	CT  *ComplexTerm
	L   List(by elements)
	HT  List(by head-tail)
	And ConjGoal
	Eq  =(Match)
	Or  DisjGoal
	R   Rule
*/

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

/* Conjunction goals */

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
		buf.WriteString(fmt.Sprint(g))
	}
	return buf.String()
}

func (cg ConjGoal) GoalType() int {
	return gtConj
}

/* Disjunction goals */

type DisjGoal []Goal

func (dg DisjGoal) GoalType() int {
	return gtDisj
}

func Or(goals ...Goal) DisjGoal {
	return DisjGoal(goals)
}

/* simple ComplexTerm goals */

func (cg *ComplexTerm) GoalType() int {
	return gtComplex
}

/* Term match goals */

type MatchGoal struct {
	L, R Term
}

type Rule struct {
	Head *ComplexTerm
	Body Goal
}

// R constructs a rule with head and a conjuction goal as the body.
func R(head *ComplexTerm, goals ...Goal) Rule {
	switch len(goals) {
	case 0:
		return Rule{Head: head}
		
	case 1:
		return Rule{Head: head, Body: goals[0]}
	}
	return Rule{Head: head, Body: ConjGoal(goals)}
}

func (r Rule) String() string {
	var buf bytes.Buffer
	buf.WriteString(r.Head.String())
	if r.Body != nil {
		buf.WriteString(" :- \n")
		buf.WriteString(appendIndent(fmt.Sprint(r.Body), "    "))
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
	fmt.Println(appendIndent(fmt.Sprint(rule), "    ") + "\n")
}

const S_FACT = " FACT "

type Context map[Variable]Term

func isSharedVar(v Variable, sCtx Context) bool {
	_, ok := sCtx[v]
	return ok
}

func setVar(v Variable, vl Term, ctx, sCtx Context) {
	if isSharedVar(v, sCtx) {
		sCtx[v] = vl
	} else {
		ctx[v] = vl
	}
}

func genSharedVar(sCtx Context) Variable {
	v := genUniqueVar()
	sCtx[v] = nil
	return v
}

// t has been unifyVar before calling
func shareTerm(t Term, ctx, sCtx Context) (cT Term) {
	switch t.Type() {
	case ttVar:
		v := t.(Variable)
		if isSharedVar(v, sCtx) {
			// v has been a shared variable
			return t
		}

		// otherwise, share it
		sV := genSharedVar(sCtx)
		ctx[v] = sV
		return sV

	case ttComplex:
		ct := t.(*ComplexTerm)
		gArgs := make([]Term, len(ct.Args))
		for i, arg := range ct.Args {
			arg = unifyVar(arg, ctx, sCtx)
			gArgs[i] = shareTerm(arg, ctx, sCtx)
		}
		return &ComplexTerm{Functor: ct.Functor, Args: gArgs}
		
	case ttList:
		switch l := t.(type) {
		case List:
			newL := make(List, len(l))
			for i, el := range l {
				el = unifyVar(el, ctx, sCtx)
				newL[i] = shareTerm(el, ctx, sCtx)
			}
			return newL
			
		case HeadTail:
			head := unifyVar(l.Head, ctx, sCtx)
			head = shareTerm(head, ctx, sCtx)
			tail := unifyVar(l.Tail, ctx, sCtx)
			tail = shareTerm(tail, ctx, sCtx)
			
			return HeadTail{Head: head, Tail: tail}
		}
	}
	return t
}

func unifyVar(t Term, ctx, sCtx Context) Term {
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

func unify(t Term, ctx, sCtx Context) Term {
	t = unifyVar(t, ctx, sCtx)
	switch t.Type() {
	case ttComplex:
		ct := t.(*ComplexTerm)
		newArgs := make([]Term, len(ct.Args))
		for i, arg := range ct.Args {
			newArgs[i] = unify(arg, ctx, sCtx)
		}
		return &ComplexTerm{Functor: ct.Functor, Args: newArgs}
	case ttList:
		switch l := t.(type) {
		case List:
			newL := make(List, len(l))
			for i, el := range l {
				newL[i] = unify(el, ctx, sCtx)
			}
			return newL
			
		case HeadTail:
			head := unify(l.Head, ctx, sCtx)
			tail := unify(l.Tail, ctx, sCtx)
			if tl, ok := tail.(List); ok {
				return append(List{head}, tl...)
			}
			return HeadTail{Head: head, Tail: tail}
		}
	}

	return t
}

func sameVar(a, b Variable, sCtx Context) bool {
	if _, ok := sCtx[a]; !ok {
		return false
	}
	if _, ok := sCtx[b]; !ok {
		return false
	}

	return a == b
}

func matchTerm(L, R Term, lCtx, rCtx, sCtx Context) (succ bool) {
	if L.Type() == ttVar {
		L = unifyVar(L, lCtx, sCtx)
	}
	if R.Type() == ttVar {
		R = unifyVar(R, rCtx, sCtx)
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
			if !sameVar(lV, rV, sCtx) {
				sV := genSharedVar(sCtx)
				setVar(lV, sV, lCtx, sCtx)
				setVar(rV, sV, rCtx, sCtx)
				// Otherwise already matche
			}
		} else {
			// L <= R
			rT := shareTerm(R, rCtx, sCtx)
			setVar(lV, rT, lCtx, sCtx)
		}
		return true
	}

	if R.Type() == ttVar {
		// L => R
		rV := R.(Variable)
		lT := shareTerm(L, lCtx, sCtx)
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
	
	// list
	if L.Type() == ttList && R.Type() == ttList {
		switch l := L.(type) {
		case List:
			switch r := R.(type) {
			case List:
				// List = List
				if len(l) != len(r) {
					return false
				}
				
				for i, lEl := range l {
					rEl := r[i]
					if !matchTerm(lEl, rEl, lCtx, rCtx, sCtx) {
						return false
					}
				}
				
				return true
				
			case HeadTail:
				if len(l) == 0 {
					return false
				}
				
				if !matchTerm(l[0], r.Head, lCtx, rCtx, sCtx) {
					return false
				}
				
				if !matchTerm(l[1:], r.Tail, lCtx, rCtx, sCtx) {
					return false
				}
				
				return true
			}
		case HeadTail:
			switch r := R.(type) {
			case List:
				if len(r) == 0 {
					return false
				}
				
				if !matchTerm(l.Head, r[0], lCtx, rCtx, sCtx) {
					return false
				}
				
				if !matchTerm(l.Tail, r[1:], lCtx, rCtx, sCtx) {
					return false
				}
				
				return true
				
			case HeadTail:
				if !matchTerm(l.Head, r.Head, lCtx, rCtx, sCtx) {
					return false
				}
				if !matchTerm(l.Tail, r.Tail, lCtx, rCtx, sCtx) {
					return false
				}
				
				return true
			}
		}
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
	//fmt.Println(indent, "rawHead", formCtx, ruleCtx);

	for v, vl := range ruleCtx {
		ruleCtx[v] = unify(vl, ruleCtx, sCtx)
	}
	for v, vl := range formCtx {
		formCtx[v] = unify(vl, formCtx, sCtx)
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

// proveGoal tries prove the goal under curtain context, push the solutions to the channel.
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
		ct = unify(ct, ctx, nil).(*ComplexTerm)

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
		ctx[v] = unify(vl, cCtx, nil)
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
