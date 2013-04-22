package prolog

import (
	"bytes"
	"fmt"
)

/*
	Short functions:
	A   Atom(by string)
	V   Variable
	CT  *ComplexTerm
	L   List(by elements)
	HT  List(by head-tail)
	FL  Atom(by first-left)

	And ConjGoal
	Eq  =(Match)
	Or  DisjGoal
	R   Rule
	Op  Operator
*/

// Constants for Goal Types. Returned by Goal.Type
const (
	gtConj    = iota // Clause, Clause
	gtDisj           // Clause; Clause
	gtComplex        // *ComplexTerm
	gtMatch          // X = Y
	gtIf             // If()Then()Else()
	gtIs             //  X is Y
	gtOp             //  X op Y
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

func (cg *buildin2) GoalType() int {
	return gtOp
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

func (m *Machine) Prove(goal Goal) (solutions chan Bindings) {
	return m.prove(goal, nil)
}

func makeSolutions(slns ...Bindings) (solutions chan Bindings) {
	solutions = make(chan Bindings, len(slns))
	for _, sln := range slns {
		solutions <- sln
	}
	close(solutions)
	return solutions
}

func trivialSolution() (solutions chan Bindings) {
	return makeSolutions(nil)
}

// prove tries prove the goal send solution Bindings to the channel. After all
// solutions are sent, the channel is closed.
// return when all solutions are received. Often called in a go routine.
// nil solutions returned means failure.
func (m *Machine) prove(goal Goal, bds Bindings) (solutions chan Bindings) {
	// fmt.Println(indent, "prove:", bds)
	// fmt.Println(appendIndent(fmt.Sprint(goal), indent))
	switch goal.GoalType() {
	case gtConj:
		cg := goal.(ConjGoal)
		if len(cg) == 0 {
			// success
			return trivialSolution()
		}
		slns0 := m.prove(cg[0], bds)
		// fmt.Println(indent, "proved:", bds, slns0)
		// fmt.Println(appendIndent(fmt.Sprint(cg[0]), indent))
		if slns0 == nil {
			return nil
		}
		if len(cg) == 1 {
			// no need go further, if nothing left
			return slns0
		}

		solutions = make(chan Bindings)
		go func() {
			for sln0 := range slns0 {
				bds1 := bds.combine(sln0)
				slns1 := m.prove(cg[1:], bds1)
				if slns1 != nil {
					for sln1 := range slns1 {
						solutions <- sln0.combine(sln1)
					}
				}
			}
			close(solutions)
		}()
		return solutions

	case gtOp:
		bi := goal.(*buildin2)
		L, R := bi.L.unify(bds), bi.R.unify(bds)

		switch bi.Op {
		case opGt, opGe, opLt, opLe, opNe:
			// comparing operators
			if L.Type() == ttInt && R.Type() == ttInt {
				l, r := L.(Integer), R.(Integer)
				bl := false
				switch bi.Op {
				case opGt:
					bl = l > r
				case opGe:
					bl = l >= r
				case opLt:
					bl = l < r
				case opLe:
					bl = l <= r
				case opNe:
					bl = l != r
				}

				if bl {
					return trivialSolution()
				}
				return nil
			}

			return nil

		case opIs:
			r := computeTerm(R)
			newBds := make(Bindings)
			if !matchTerm(L, r, newBds) {
				return nil
			}

			// fmt.Println(indent, "opIs", bi, ",", L, "is", R, "=", r, "=>", newBds)
			return makeSolutions(newBds)
		}

		panic(fmt.Sprintf("Op %s is not a valid goal.", bi))

	case gtComplex:
		ct := goal.(*ComplexTerm)
		ct = ct.unify(bds).(*ComplexTerm)

		return m.Match(ct)

	default:
		panic(fmt.Sprintf("Goal not supported: %s", goal))
	}

	return nil
}

func calcSolution(inBds, bds Bindings) (sln Bindings) {
	sln = make(Bindings)
	for v, vl := range inBds {
		sln[v] = vl.unify(bds)
	}

	return sln
}

// for debugging
var indent string

func (m *Machine) Match(query *ComplexTerm) (solutions chan Bindings) {
	inBds := make(Bindings)
	// localized query
	lq := query.repQueryVars(inBds).(*ComplexTerm)
	// fmt.Println(indent, "Match:", query, lq)
	//indent += "    "
	//defer func() { indent = indent[:len(indent)-4] }()

	solutions = make(chan Bindings)

	go func() {
		rules := m.rules[query.Key()]
		for _, rule := range rules {
			hdBds := matchHead(rule.Head, lq)
			if hdBds == nil {
				// head not matched
				continue
			}
			// fmt.Println(indent, "Head", lq, rule.Head, hdBds)

			if rule.Body == nil {
				// For a head-matched fact, generate a single solution.
				// fmt.Println(indent, lq, "Fact", rule.Head, hdBds)
				solutions <- calcSolution(inBds, hdBds)
				//break
				continue
			}

			slns := m.prove(rule.Body, hdBds)
			if slns != nil {
				for sln := range slns {
					// fmt.Println(indent, "sln:", sln, hdBds)
					solutions <- calcSolution(inBds, hdBds.combine(sln))
				}
			}
		}
		close(solutions)
	}()

	return solutions
}

func NewMachine() *Machine {
	return &Machine{rules: map[string][]Rule{}}
}
