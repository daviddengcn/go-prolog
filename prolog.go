package prolog

import (
	"bytes"
	"fmt"
)

type Term interface {
	Match(t Term, sln Solution) (succ bool)
	ReplaceVariables(varMap map[Variable]Variable) Term
	Instantiate(sln Solution) Term
}

type Atom string

func A(str string) Atom {
	return Atom(str)
}

func (at Atom) Match(t Term, sln Solution) (succ bool) {
	//fmt.Println("Atom", at, "Matching(", t, ")", sln)
	insT := t.Instantiate(sln)
	switch mt := insT.(type) {
	case Atom:
		succ = at == mt
	case Variable:
		sln[mt] = at
		succ = true
	default:
		succ = false
	}
	//fmt.Println("Atom", at, "Match(", t, ")", sln, ", ", succ)
	return succ
}

func (at Atom) Instantiate(sln Solution) Term {
	return at
}

func (at Atom) ReplaceVariables(varMap map[Variable]Variable) Term {
	return at
}

type Variable string

func (v Variable) Match(t Term, sln Solution) (succ bool) {
	//fmt.Println("Variable", v, "Matching(", t, ")", sln)
	ins := v.Instantiate(sln)
	switch vv := ins.(type) {
	case Variable:
		succ = true
		switch mt := t.(type) {
		case Variable:
			tt := mt.Instantiate(sln)
			if tt == vv {
			} else {
				sln[vv] = tt
			}
		default:
			succ = true
			sln[vv] = mt.Instantiate(sln)
		}
	default:
		succ = ins.Match(t, sln)
	}

	//fmt.Println("Variable", v, "Match(", t, ")", sln, ", ", succ)
	return succ
}

func (v Variable) Instantiate(sln Solution) Term {
	ins, ok := sln[v]
	if ok {
		return ins
	}
	return v
}

var gUniqueVarChan chan Variable

func init() {
	gUniqueVarChan = make(chan Variable, 10)
	go func() {
		counter := 0
		for {
			gUniqueVarChan <- V(fmt.Sprintf("_AUTO_%d", counter))
			counter++
		}
	}()
}

func genUniqueVar() Variable {
	return <-gUniqueVarChan
}

// varMap: v -> newV
func (v Variable) ReplaceVariables(varMap map[Variable]Variable) Term {
	newV, ok := varMap[v]
	if !ok {
		newV := genUniqueVar()
		varMap[v] = newV
		return newV
	}

	return newV
}

func V(str string) Variable {
	return Variable(str)
}

type ComplexTerm struct {
	Functor Atom
	Args    []Term
}

func NewComplexTerm(functor Atom, args ...Term) *ComplexTerm {
	return &ComplexTerm{Functor: functor, Args: args}
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

func (ct *ComplexTerm) Match(t Term, sln Solution) (succ bool) {
	switch mt := t.(type) {
	case Variable:
		mIns, ok := sln[mt]
		if !ok {
			sln[mt] = ct
			succ = true
		} else {
			succ = ct.Match(mIns, sln)
		}

	case *ComplexTerm:
		if ct.Functor == mt.Functor && len(ct.Args) == len(mt.Args) {
			succ = true
			for i, arg := range ct.Args {
				succ = arg.Match(mt.Args[i], sln)
				if !succ {
					break
				}
			}
		} else {
			succ = false
		}
	default:
		succ = false
	}

	//fmt.Println(ct, "Match(", t, ", ", sln, ") ", succ)

	return succ
}

func (ct *ComplexTerm) Instantiate(sln Solution) Term {
	newArgs := make([]Term, len(ct.Args))
	for i, arg := range ct.Args {
		newArgs[i] = arg.Instantiate(sln)
	}

	return NewComplexTerm(ct.Functor, newArgs...)
}

func (ct *ComplexTerm) ReplaceVariables(varMap map[Variable]Variable) Term {
	newArgs := make([]Term, len(ct.Args))
	for i, arg := range ct.Args {
		newArgs[i] = arg.ReplaceVariables(varMap)
	}

	return NewComplexTerm(ct.Functor, newArgs...)
}

func (ct *ComplexTerm) AllVariables(st map[Variable]struct{}) {
	for _, arg := range ct.Args {
		switch a := arg.(type) {
		case Variable:
			st[a] = struct{}{}

		case *ComplexTerm:
			a.AllVariables(st)
		}
	}
}

type Solution map[Variable]Term

/*****************
	Machine Type
*****************/

type Machine struct {
	facts map[string][]*ComplexTerm
}

func (m *Machine) AddFact(head *ComplexTerm) {
	key := head.Key()
	m.facts[key] = append(m.facts[key], head.ReplaceVariables(make(map[Variable]Variable)).(*ComplexTerm))
	fmt.Println("Adding fact:", head)
}

const S_FACT = " FACT "

func (m *Machine) MatchFact(head *ComplexTerm, solutions chan Solution) {
	//fmt.Println("Matching fact:", head)

	//fmt.Println("varMap", varMap, repHead)
	vars := make(map[Variable]struct{})
	head.AllVariables(vars)

	facts := m.facts[head.Key()]
	for _, fact := range facts {
		sln := make(Solution)
		if head.Match(fact, sln) {
			s := make(Solution)
			for v := range vars {
				ins, ok := sln[v]
				if ok {
					s[v] = ins.Instantiate(sln)
				}
			}
			s[S_FACT] = Atom(fact.String())
			solutions <- s
		}
	}
	close(solutions)
}

func NewMachine() *Machine {
	return &Machine{facts: map[string][]*ComplexTerm{}}
}
