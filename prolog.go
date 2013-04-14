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

type Bindings map[Variable]Term

func (bds Bindings) unifyVar(t Term) Term {
	for t.Type() == ttVar {
		v := t.(Variable)
		i := bds[v];
		if i == nil {
			break
		}
		t = i
	}
	return t
}

func (bds Bindings) unify(t Term) Term {
	t = bds.unifyVar(t)
	switch t.Type() {
	case ttComplex:
		ct := t.(*ComplexTerm)
		newArgs := make([]Term, len(ct.Args))
		for i, arg := range ct.Args {
			newArgs[i] = bds.unify(arg)
		}
		return &ComplexTerm{Functor: ct.Functor, Args: newArgs}
	case ttList:
		switch l := t.(type) {
		case List:
			newL := make(List, len(l))
			for i, el := range l {
				newL[i] = bds.unify(el)
			}
			return newL
			
		case HeadTail:
			head := bds.unify(l.Head)
			tail := bds.unify(l.Tail)
			if tl, ok := tail.(List); ok {
				return append(List{head}, tl...)
			}
			return HeadTail{Head: head, Tail: tail}
		}
	}

	return t
}

func matchTerm(L, R Term, bds Bindings) (succ bool) {
	if L.Type() == ttVar {
		L = bds.unifyVar(L)
	}
	if R.Type() == ttVar {
		R = bds.unifyVar(R)
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
				bds[lV] = sV
				bds[rV] = sV
				// Otherwise already matche
			}
		} else {
			// lV <= R
			bds[lV] = R
		}
		return true
	}

	if R.Type() == ttVar {
		// L => rV
		rV := R.(Variable)
		bds[rV] = L
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
			if !matchTerm(lArg, rArg, bds) {
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
					if !matchTerm(lEl, rEl, bds) {
						return false
					}
				}
				
				return true
				
			case HeadTail:
				if len(l) == 0 {
					return false
				}
				
				if !matchTerm(l[0], r.Head, bds) {
					return false
				}
				
				if !matchTerm(l[1:], r.Tail, bds) {
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
				
				if !matchTerm(l.Head, r[0], bds) {
					return false
				}
				
				if !matchTerm(l.Tail, r[1:], bds) {
					return false
				}
				
				return true
				
			case HeadTail:
				if !matchTerm(l.Head, r.Head, bds) {
					return false
				}
				if !matchTerm(l.Tail, r.Tail, bds) {
					return false
				}
				
				return true
			}
		}
	}

	return false
}

func matchHead(rule, q *ComplexTerm) (bds Bindings) {
	bds = make(Bindings)
	for i, ruleArg := range rule.Args {
		qArg := q.Args[i]
		if !matchTerm(ruleArg, qArg, bds) {
			return nil
		}
	}

	return bds
}

func mulContext(b, s Bindings) Bindings {
	c := make(Bindings)
	for v, vl := range b {
		c[v] = vl
	}
	for v, vl := range s {
		c[v] = vl
	}

	return c
}

// proveGoal tries prove the goal under curtain context, push the solutions to the channel.
func (m *Machine) proveGoal(goal Goal, bds Bindings, solutions chan Bindings) {
	switch goal.GoalType() {
	case gtConj:
		cg := goal.(ConjGoal)
		if len(cg) == 0 {
			solutions <- nil
			close(solutions)
			return
		}
		g := cg[0]
		slns := make(chan Bindings)
		go m.proveGoal(g, bds, slns)
		for dctx := range slns {
			ctx2 := mulContext(bds, dctx)
			slns2 := make(chan Bindings)
			go m.proveGoal(cg[1:], ctx2, slns2)
			for dctx2 := range slns2 {
				sln3 := mulContext(ctx2, dctx2)
				solutions <- sln3
			}
		}
		close(solutions)

	case gtComplex:
		ct := goal.(*ComplexTerm)
		ct = bds.unify(ct).(*ComplexTerm)

		slns := make(chan Bindings)
		go m.Match(ct, slns)
		for dctx := range slns {
			//fmt.Println(indent, "G", goal, ctx, ct, dctx)
			ctx2 := mulContext(bds, dctx)
			solutions <- ctx2
		}
		close(solutions)

	default:
		panic(fmt.Sprintf("Goal not supported: %s", goal))
	}
}

func calcSolution(inBds, mBds Bindings) (sln Bindings) {
	sln = make(Bindings)
	for v, vl := range inBds {
		sln[v] = mBds.unify(vl.(Variable))
	}
	
	return sln
}

// for debugging
var indent string

func (m *Machine) Match(query *ComplexTerm, solutions chan Bindings) {
	inBds := make(Bindings)
	// localized query
	lq := query.repQueryVars(inBds).(*ComplexTerm)
	fmt.Println(indent, "Match:", query, lq)
	indent += "    "
	defer func() { indent = indent[:len(indent)-4] }()

	rules := m.rules[query.Key()]
	for _, rule := range rules {
		hdBds := matchHead(rule.Head, lq)
		if hdBds == nil {
			// head not matched
			continue
		}
		//fmt.Println(indent, "Head", form, formCtx, rule.Head, ruleCtx)

		if rule.Body == nil {
			//fmt.Println(indent, form, "Fact", rule.Head, formCtx)
			solutions <- calcSolution(inBds, hdBds)
			continue
		}

		slns := make(chan Bindings)
		go m.proveGoal(rule.Body, hdBds, slns)
		for dctx := range slns {
			//fmt.Println(indent, "sln:", sln, mFormHead)
			solutions <- calcSolution(inBds, dctx)
		}
	}
	close(solutions)
}

func NewMachine() *Machine {
	return &Machine{rules: map[string][]Rule{}}
}
 