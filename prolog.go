package plg

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
	replaceGoalVars(bds VarBindings) Goal

	// at most one solution
	singleSolution() bool
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

func (cg ConjGoal) replaceGoalVars(bds VarBindings) Goal {
	newCg := make(ConjGoal, len(cg))
	for i, g := range cg {
		newCg[i] = g.replaceGoalVars(bds)
	}

	return newCg
}

func (cg ConjGoal) singleSolution() bool {
	for _, g := range cg {
		if !g.singleSolution() {
			return false
		}
	}

	return true
}

/* Disjunction goals */

type DisjGoal []Goal

func Or(goals ...Goal) DisjGoal {
	return DisjGoal(goals)
}

func (dg DisjGoal) GoalType() int {
	return gtDisj
}

func (dg DisjGoal) replaceGoalVars(bds VarBindings) Goal {
	newDg := make(DisjGoal, len(dg))
	for i, g := range dg {
		newDg[i] = g.replaceGoalVars(bds)
	}

	return newDg
}

func (dg DisjGoal) singleSolution() bool {
	if len(dg) == 0 {
		return true
	}

	if len(dg) == 1 {
		return dg[0].singleSolution()
	}

	return false
}

/* simple ComplexTerm goals */

func (ct *ComplexTerm) GoalType() int {
	return gtComplex
}

func (ct *ComplexTerm) replaceGoalVars(bds VarBindings) Goal {
	newCt := &ComplexTerm{Functor: ct.Functor, Args: make([]Term, len(ct.Args))}
	for i, arg := range ct.Args {
		newCt.Args[i] = arg.replaceVars(bds)
	}
	return newCt
}

func (ct *ComplexTerm) singleSolution() bool {
	return false
}

func (b *buildin2) GoalType() int {
	return gtOp
}

func (bi *buildin2) replaceGoalVars(bds VarBindings) Goal {
	return &buildin2{Op: bi.Op,
		L: bi.L.replaceVars(bds), R: bi.R.replaceVars(bds)}
}

func (b *buildin2) singleSolution() bool {
	return true
}

/* Term match goals */

type MatchGoal struct {
	L, R Term
}

type Rule struct {
	Head *ComplexTerm
	Body Goal
	vBds rVarBindings
}

func (r Rule) RVarCount() int {
	return len(r.vBds)
}

// R constructs a rule with head and a conjuction goal as the body.
func R(head *ComplexTerm, goals ...Goal) *Rule {
	switch len(goals) {
	case 0:
		return &Rule{Head: head}

	case 1:
		return &Rule{Head: head, Body: goals[0]}
	}
	return &Rule{Head: head, Body: ConjGoal(goals)}
}

func (r *Rule) String() string {
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
	rules map[int][]*Rule
}

func (m *Machine) AddFact(head *ComplexTerm) {
	m.AddRule(&Rule{Head: head})
}

func (m *Machine) AddRule(rule *Rule) {
	fmt.Println(appendIndent(fmt.Sprint(rule), "    ") + "\n")

	key := rule.Head.Key()
	m.rules[key] = append(m.rules[key], rule)

	bds := make(rVarBindings)
	rule.Head = rule.Head.replaceVars(bds).(*ComplexTerm)
	if rule.Body != nil {
		rule.Body = rule.Body.replaceGoalVars(bds)
	}
	rule.vBds = bds
	fmt.Println("Replaced:", appendIndent(fmt.Sprint(rule), "    ")+"\n")
}

// returns nil if not matched
func (r Rule) matchHead(q *ComplexTerm) *Bindings {
	bds := newBindings(nil, r.RVarCount())
	for i, headArg := range r.Head.Args {
		qArg := q.Args[i]
		if !matchTerm(headArg, qArg, bds) {
			return nil
		}
	}

	return bds
}

func (m *Machine) Prove(goal Goal) (solutions chan *Bindings) {
	return m.prove(goal, nil)
}

func makeSolutions(slns ...*Bindings) (solutions chan *Bindings) {
	solutions = make(chan *Bindings, len(slns))
	for _, sln := range slns {
		solutions <- sln
	}
	close(solutions)
	return solutions
}

func (m *Machine) process(goal Goal, bds *Bindings) bool {
	switch goal.GoalType() {
	case gtConj:
		cg := goal.(ConjGoal)
		if len(cg) == 0 {
			// success
			return true
		}

		for _, g := range cg {
			if !m.process(g, bds) {
				return false
			}
		}
		return true

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
					return true
				}
				return false
			}

			return false

		case opIs:
			r := computeTerm(R)
			if !matchTerm(L, r, bds) {
				return false
			}

			// fmt.Println(indent, "opIs", bi, ",", L, "is", R, "=", r, "=>", newBds)
			return true
		}

		panic(fmt.Sprintf("Op %s is not a valid goal.", bi))

	}

	panic(fmt.Sprint(goal) + " is not singleSolution!")
}

// prove tries prove the goal send solution Bindings to the channel. After all
// solutions are sent, the channel is closed.
// return when all solutions are received. Often called in a go routine.
// nil solutions returned means failure.
//
// bds: may be changed (put new bindings), the caller should solve the reuse
//      problem. The caller should not change it after returned.
// solution: all bindings along with new bindings, i.e. bds + new bindgs, this value
//           will not be modified later, so can be referenced/modified safely.
func (m *Machine) prove(goal Goal, bds *Bindings) (solutions chan *Bindings) {
	// fmt.Println(indent, "prove:", bds)
	// fmt.Println(appendIndent(fmt.Sprint(goal), indent))
	switch goal.GoalType() {
	case gtConj:
		cg := goal.(ConjGoal)
		start := 0
		for start < len(cg) && cg[start].singleSolution() {
			if !m.process(cg[start], bds) {
				return nil
			}

			start++
		}
		if start == len(cg) {
			// success
			return makeSolutions(bds)
		}

		slns0 := m.prove(cg[start], bds)
		start++
		// fmt.Println(indent, "proved:", bds, slns0)
		// fmt.Println(appendIndent(fmt.Sprint(cg[0]), indent))
		if slns0 == nil {
			return nil
		}
		if start == len(cg) {
			// no need go further, if nothing left
			return slns0
		}

		solutions = make(chan *Bindings)
		go func() {
			remains := cg[start:]
			for sln0 := range slns0 {
				slns1 := m.prove(remains, newBindingsFrom(sln0))
				if slns1 != nil {
					for sln1 := range slns1 {
						solutions <- sln1
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
					return makeSolutions(bds)
				}
				return nil
			}

			return nil

		case opIs:
			r := computeTerm(R)
			if !matchTerm(L, r, bds) {
				return nil
			}

			// fmt.Println(indent, "opIs", bi, ",", L, "is", R, "=", r, "=>", newBds)
			return makeSolutions(bds)
		}

		panic(fmt.Sprintf("Op %s is not a valid goal.", bi))

	case gtComplex:
		ct := goal.(*ComplexTerm)
		ct = ct.unify(bds).(*ComplexTerm)

		return m.match(ct, bds)

	default:
		panic(fmt.Sprintf("Goal not supported: %s", goal))
	}

	return nil
}

func calcSolution(qBds *Bindings, inBds *pVarBindings, bds *Bindings) (sln *Bindings) {
	sln = newBindingsFrom(qBds)
	inBds.each(func(v, vl variable) {
		sln.Put(v, vl.export(bds))
	})

	return sln
}

// for debugging
var indent string

func (m *Machine) Match(query *ComplexTerm) (solutions chan *Bindings) {
	return m.match(query, nil)
}

// query: has been unified
// qBds: Bings base of query
// solution: gV/rV -> const/gV
func (m *Machine) match(query *ComplexTerm, qBds *Bindings) (solutions chan *Bindings) {
	/* localized query: query -> lq */
	// query.gV/rV -> pVas
	inBds := newPVarBindings(qBds.RVarCount())

	lq := query.replaceVars(inBds).(*ComplexTerm)
	//fmt.Println(indent, "replaceVars", query, "->", lq)
	//indent += "    "
	//defer func() { indent = indent[:len(indent)-4] }()

	// each solution: query.g/rVars -> const, gVars
	solutions = make(chan *Bindings)

	go func() {
		rules := m.rules[query.Key()]
		for _, rule := range rules {
			hdBds := rule.matchHead(lq)
			if hdBds == nil {
				// head not matched
				continue
			}
			//fmt.Println(indent, "Head matched:", lq, "<->", rule.Head, "under", hdBds)

			if rule.Body == nil {
				// For a head-matched fact, generate a single solution.
				//fmt.Println(indent, lq, "Fact", rule.Head, hdBds)
				solutions <- calcSolution(qBds, inBds, hdBds)
				//break
				continue
			}

			slns := m.prove(rule.Body, hdBds)
			if slns != nil {
				for sln := range slns {
					// fmt.Println(indent, "sln:", sln, hdBds)
					solutions <- calcSolution(qBds, inBds, sln)
				}
			}
		}
		close(solutions)
	}()

	return solutions
}

func NewMachine() *Machine {
	return &Machine{rules: make(map[int][]*Rule)}
}
