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
	gtIf      // If()Then()Else()
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

// returns nil if not matched
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

// prove tries prove the goal send solution Bindings to the channel. After all
// solutions are sent, the channel is closed.
// return when all solutions are received. Often called in a go routine.
func (m *Machine) prove(goal Goal, bds Bindings, solutions chan Bindings) {
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
		go m.prove(g, bds, slns)
		for sln := range slns {
			bds2 := bds.combine(sln)
			slns2 := make(chan Bindings)
			go m.prove(cg[1:], bds2, slns2)
			for sln2 := range slns2 {
				solutions <- sln.combine(sln2)
			}
		}
		close(solutions)

	case gtComplex:
		ct := goal.(*ComplexTerm)
		ct = bds.unify(ct).(*ComplexTerm)

		slns := make(chan Bindings)
		go m.Match(ct, slns)
		for dctx := range slns {
//			fmt.Println(indent, "G CT", goal, bds, ct, dctx)
			solutions <- dctx
		}
		close(solutions)

	default:
		panic(fmt.Sprintf("Goal not supported: %s", goal))
	}
}

func calcSolution(inBds, bds Bindings) (sln Bindings) {
	sln = make(Bindings)
	for v, vl := range inBds {
		sln[v] = bds.unify(vl)
	}
	
	return sln
}

// for debugging
var indent string

func (m *Machine) Match(query *ComplexTerm, solutions chan Bindings) {
	inBds := make(Bindings)
	// localized query
	lq := query.repQueryVars(inBds).(*ComplexTerm)
	//fmt.Println(indent, "Match:", query, lq)
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
		go m.prove(rule.Body, hdBds, slns)
		for sln := range slns {
			//fmt.Println(indent, "sln:", sln, mFormHead)
			solutions <- calcSolution(inBds, hdBds.combine(sln))
		}
	}
	close(solutions)
}

func NewMachine() *Machine {
	return &Machine{rules: map[string][]Rule{}}
}
