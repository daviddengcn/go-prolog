package prolog

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// Constants for Term.Type().
const (
	ttAtom    = iota // Atom, FirstLeft
	ttInt            // Integer
	ttVar            // Variable
	ttComplex        // *ComplexTerm
	ttList           // List, HeadTail
)

type Term interface {
	Type() int
	// replace query variabls
	repQueryVars(bds Bindings) (newT Term)

	// l: the receiver
	// l and R has be unifyVar before called
	// if l is not Variable, R is not Variable
	Match(R Term, bds Bindings) bool

	unify(bds Bindings) Term
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
	case int:
		return Integer(vl)
	case int32:
		return Integer(vl)
	case int64:
		return Integer(vl)
		
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

func (at Atom) String() string {
	if len(at) > 0 {
		if at[0] >= '0' && at[0] <= '9' {
			return strconv.Quote(string(at))
		}
	}
	
	return string(at)
}

func (at Atom) Type() int {
	return ttAtom
}

func (at Atom) repQueryVars(bds Bindings) Term {
	return at
}

func (l Atom) Match(R Term, bds Bindings) bool {
	if r, ok := R.(Atom); ok {
		return l == r
	}

	return R.Match(l, bds)
}

func (at Atom) unify(bds Bindings) Term {
	return at
}

/* Integer numbers: Integer */

type Integer int

func I(i int) Integer {
	return Integer(i)
}

func (i Integer) Type() int {
	return ttInt
}

func (i Integer) repQueryVars(bds Bindings) Term {
	return i
}

func (l Integer) Match(R Term, bds Bindings) bool {
	if R.Type() == ttAtom {
		return false
	}
	
	if r, ok := R.(Integer); ok {
		return l == r
	}
	
	return R.Match(l, bds)
}

func (i Integer) unify(bds Bindings) Term {
	return i
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

func (l Variable) Match(R Term, bds Bindings) bool {
	if R.Type() == ttVar {
		// both Variable's
		r := R.(Variable)
		if l != r {
			s := genUniqueVar()
			bds[l] = s
			bds[r] = s
			// Otherwise already matche
		}
	} else {
		// lV <= R
		bds[l] = R
	}
	return true
}

func (v Variable) unify(bds Bindings) Term {
	t := bds.unifyVar(v)
	if t.Type() != ttVar {
		return t.unify(bds)
	}

	return t
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

func (l *ComplexTerm) Match(R Term, bds Bindings) bool {
	if R.Type() != ttComplex {
		return false
	}

	r := R.(*ComplexTerm)
	if l.Functor != r.Functor || len(l.Args) != len(r.Args) {
		return false
	}

	for i, lArg := range l.Args {
		rArg := r.Args[i]
		if !matchTerm(lArg, rArg, bds) {
			return false
		}
	}

	return true
}

func (ct *ComplexTerm) unify(bds Bindings) Term {
	newArgs := make([]Term, len(ct.Args))
	for i, arg := range ct.Args {
		newArgs[i] = arg.unify(bds)
	}
	return &ComplexTerm{Functor: ct.Functor, Args: newArgs}
}

/* List term: List */
type List []Term

func L(terms ...interface{}) (l List) {
	l = make(List, len(terms))
	for i, t := range terms {
		l[i] = term(t)
	}

	return l
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

func (l List) Match(R Term, bds Bindings) bool {
	if R.Type() != ttList {
		return false
	}

	r, ok := R.(List)
	if !ok {
		return R.Match(l, bds)
	}

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
}

func (l List) unify(bds Bindings) Term {
	newL := make(List, len(l))
	for i, el := range l {
		newL[i] = el.unify(bds)
	}
	return newL
}

/*
	List represented as [Head|Tail]: HeadTail
	HeadTail does not directly support [X, Y|Z], use [X|[Y|Z]] instead.
*/
type HeadTail struct {
	Head Term
	Tail Term
}

func HT(head, tail interface{}) HeadTail {
	return HeadTail{Head: term(head), Tail: term(tail)}
}

func (l HeadTail) Type() int {
	return ttList
}

func (l HeadTail) String() string {
	return fmt.Sprintf("[%s|%s]", l.Head, l.Tail)
}

func (l HeadTail) repQueryVars(bds Bindings) Term {
	return HeadTail{Head: l.Head.repQueryVars(bds),
		Tail: l.Tail.repQueryVars(bds)}
}

func (l HeadTail) Match(R Term, bds Bindings) bool {
	if R.Type() != ttList {
		return false
	}

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

	case HeadTail:
		if !matchTerm(l.Head, r.Head, bds) {
			return false
		}
		if !matchTerm(l.Tail, r.Tail, bds) {
			return false
		}

	}

	return true
}

func (l HeadTail) unify(bds Bindings) Term {
	head := l.Head.unify(bds)
	tail := l.Tail.unify(bds)
	if tl, ok := tail.(List); ok {
		// merge back to List
		return append(List{head}, tl...)
	}
	return HeadTail{Head: head, Tail: tail}
}

/* First-left atom: FirstLeft */
type FirstLeft struct {
	First, Left Term
}

func FL(first, left interface{}) FirstLeft {
	return FirstLeft{First: term(first), Left: term(left)}
}

func (at FirstLeft) Type() int {
	return ttAtom
}

func (at FirstLeft) String() string {
	return fmt.Sprintf("%s+%s", at.First, at.Left)
}

func (at FirstLeft) repQueryVars(bds Bindings) Term {
	return FirstLeft{First: at.First.repQueryVars(bds),
		Left: at.Left.repQueryVars(bds)}
}

func (l FirstLeft) Match(R Term, bds Bindings) bool {
	if R.Type() != ttAtom {
		return false
	}

	switch r := R.(type) {
	case Atom:
		if len(r) < 1 {
			return false
		}

		if !matchTerm(l.First, Atom(r[0:1]), bds) {
			return false
		}
		if !matchTerm(l.Left, Atom(r[1:]), bds) {
			return false
		}
	case FirstLeft:
		if !matchTerm(l.First, r.First, bds) {
			return false
		}
		if !matchTerm(l.Left, r.Left, bds) {
			return false
		}
	}

	return true
}

func (at FirstLeft) unify(bds Bindings) Term {
	first := at.First.unify(bds)
	left := at.Left.unify(bds)

	if fst, ok := first.(Atom); ok {
		if lft, ok := left.(Atom); ok {
			return fst + lft
		}
	}
	return FirstLeft{First: first, Left: left}
}

/* Bindings: Variable map to its value as a Term */

type Bindings map[Variable]Term

// keep unify until t is no longer a Variable, but no further unify
func (bds Bindings) unifyVar(t Term) Term {
	for t.Type() == ttVar {
		v := t.(Variable)
		i := bds[v]
		if i == nil {
			break
		}
		t = i
	}
	return t
}

// c = a + b...
func (a Bindings) combine(bs ...Bindings) (c Bindings) {
	c = make(Bindings)

	for v, vl := range a {
		c[v] = vl
	}

	for _, b := range bs {
		for v, vl := range b {
			c[v] = vl
		}
	}

	return c
}

func matchTerm(L, R Term, bds Bindings) (succ bool) {
	if L.Type() == ttVar {
		L = bds.unifyVar(L)
	}
	if R.Type() == ttVar {
		R = bds.unifyVar(R)
	}

	if L.Type() == ttVar {
		return L.Match(R, bds)
	}

	if R.Type() == ttVar {
		return R.Match(L, bds)
	}

	if L.Type() < R.Type() {
		return L.Match(R, bds)
	}

	return R.Match(L, bds)
}
