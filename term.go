package prolog

import (
	"bytes"
	"fmt"
	"strings"
)

// Constants for Term Types.
const (
	ttAtom = iota
	ttVar
	ttComplex
	ttList
)

type Term interface {
	Type() int
	repQueryVars(bds Bindings) (newT Term)
}

func isPrologVariableStart(c byte) bool {
	return c == '_' || c >= 'A' && c <= 'Z'
}

func TermFromString(s string) Term {
	if len(s) > 0 && isPrologVariableStart(s[0]) {
		return V(s)
	}

	return A(s)
}

func term(t interface{}) Term {
	switch vl := t.(type) {
	case string:
		return TermFromString(vl)

	case Term:
		return vl
	}
	panic(fmt.Sprintf("Invalid argument %v for CT", t))
}

/* Atom term: Atom */

type Atom string

func A(str string) Atom {
	return Atom(str)
}

func (at Atom) Type() int {
	return ttAtom
}

func (at Atom) repQueryVars(bds Bindings) Term {
	return at
}

/* Variable term: Variable */

type Variable string

const _VAR_GLOBAL_PREFIX = "_AUTO_"
const _VAR_GLOBAL_FMT = "_AUTO_%d"

func V(str string) Variable {
	if strings.HasPrefix(str, _VAR_GLOBAL_PREFIX) {
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

func (v Variable) repQueryVars(bds Bindings) Term {
	newV, ok := bds[v]
	if ok {
		return newV
	}
	newV = genUniqueVar()
	bds[v] = newV

	return newV
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

/* Complex term: *ComplexTerm */

type ComplexTerm struct {
	Functor Atom
	Args    []Term
}

// CT creates a *ComplexTerm instance with functor and args.
// The arg with type Term will be inserted directly, and an string will be
// converted into either an Atom or an Variable using TermFromString.
//
// NOTE: using V("v"), you can create variables with lower case prefix. This is
// legal in go-prolog.
func CT(functor Atom, args ...interface{}) *ComplexTerm {
	return &ComplexTerm{Functor: functor, Args: L(args...)}
}

func (ct *ComplexTerm) Type() int {
	return ttComplex
}

func (ct *ComplexTerm) repQueryVars(bds Bindings) Term {
	newCt := &ComplexTerm{Functor: ct.Functor, Args: make([]Term, len(ct.Args))}
	for i, arg := range ct.Args {
		newCt.Args[i] = arg.repQueryVars(bds)
	}
	return newCt
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

/* List term: List */
type List []Term

/*
	List represented as [Head|Tail]: HeadTail
	HeadTail does not directly support [X, Y|Z], use [X|[Y|Z]] instead.
*/
type HeadTail struct {
	Head Term
	Tail Term
}

func (l List) Type() int {
	return ttList
}

func (l List) repQueryVars(bds Bindings) Term {
	newL := make(List, len(l))
	for i, el := range l {
		newL[i] = el.repQueryVars(bds)
	}
	return newL
}

func L(terms ...interface{}) (l List) {
	l = make(List, len(terms))
	for i, t := range terms {
		l[i] = term(t)
	}

	return l
}

func (l HeadTail) Type() int {
	return ttList
}

func HT(head, tail interface{}) HeadTail {
	return HeadTail{Head: term(head), Tail: term(tail)}
}

func (l HeadTail) String() string {
	return fmt.Sprintf("[%s|%s]", l.Head, l.Tail)
}

func (l HeadTail) repQueryVars(bds Bindings) Term {
	return HeadTail{Head: l.Head.repQueryVars(bds),
		Tail: l.Tail.repQueryVars(bds)}
}
