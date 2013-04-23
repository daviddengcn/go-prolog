package prolog

import (
	"bytes"
	"fmt"
	"github.com/daviddengcn/go-villa"
	//	"strconv"
	//"strings"
)

// Constants for Term.Type().
const (
	ttAtom    = iota // Atom, FirstLeft
	ttInt            // Integer
	ttVar            // Variable
	ttComplex        // *ComplexTerm
	ttList           // List, HeadTail
	ttBuildin        // Buildin operators
)

type VarBindings interface {
	get(v variable) variable
}

type Term interface {
	Type() int
	// replace query variabls with p-variables
	// bds: q-var -> p-var
	// pIndex of variables in bds are from 0 - len(bds) -1, if a new variable
	// has to be generated, use pV(len(bds), then put it into bds
	replaceVars(bds VarBindings) (newT Term)
	
	// l: the receiver
	// l and R has be unifyVar before called
	// if l is not Variable, R is not Variable
	Match(R Term, bds *Bindings) bool

	unify(bds *Bindings) Term
	export(bds *Bindings) Term
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

type atom int

func A(name string) atom {
	return atom(gAtomPool.indexOfName(name))
}

var gAtomPool = newNamePool()

func (at atom) String() string {
	return gAtomPool.nameOfIndex(int(at))
}

func (at atom) Type() int {
	return ttAtom
}

func (at atom) replaceVars(bds VarBindings) Term {
	return at
}

func (l atom) Match(R Term, bds *Bindings) bool {
	if r, ok := R.(atom); ok {
		return l == r
	}

	return R.Match(l, bds)
}

func (at atom) unify(bds *Bindings) Term {
	return at
}

func (at atom) export(bds *Bindings) Term {
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

func (i Integer) replaceVars(bds VarBindings) Term {
	return i
}

func (l Integer) Match(R Term, bds *Bindings) bool {
	if R.Type() == ttAtom {
		return false
	}

	if r, ok := R.(Integer); ok {
		return l == r
	}

	return R.Match(l, bds)
}

func (i Integer) unify(bds *Bindings) Term {
	return i
}

func (i Integer) export(bds *Bindings) Term {
	return i
}

/* Variable term: Variable */

var gVarPool = newNamePool()

type variable int

func V(name string) variable {
	return variable(gVarPool.indexOfName(name))
}

// gIndex -> variable
func gV(gIndex int) variable {
	return variable(-(gIndex + 1)*4)
}
// variable -> gIndex
func (v variable) gIndex() int {
	return (-int(v))/4 - 1
}
// whether v is a g-variable
func (v variable) isG() bool {
	return v < 0 && (-int(v)) % 4 == 0
}

// pIndex -> variable
func pV(pIndex int) variable {
	return variable(-(pIndex*4 + 1))
}
// variable -> pIndex
func (v variable) pIndex() int {
	return (-int(v)) / 4
}
// whether v is a p-variable
func (v variable) isP() bool {
	return v < 0 && (-int(v)) % 4 == 1
}

// rIndex -> variable
func rV(rIndex int) variable {
	return variable(-(rIndex*4 + 2))
}
// variable -> rIndex
func (v variable) rIndex() int {
	return (-int(v)) / 4
}
// whether v is a r-variable
func (v variable) isR() bool {
	return v < 0 && (-int(v)) % 4 == 2
}

func (v variable) Type() int {
	return ttVar
}

func (v variable) String() string {
	if v >= 0 {
		return gVarPool.nameOfIndex(int(v))
	}

	if v.isG() {
		return fmt.Sprintf("G_%d", v.gIndex())
	}
	
	if v.isP() {
		return fmt.Sprintf("P_%d", v.pIndex())
	}
	
	if v.isR() {
		return fmt.Sprintf("R_%d", v.rIndex())
	}
	
	return fmt.Sprint("Invalid_%d", -v)
}

func (v variable) replaceVars(bds VarBindings) Term {
	return bds.get(v)
}

func (l variable) Match(R Term, bds *Bindings) bool {
	if R.Type() == ttVar {
		// both Variable's
		r := R.(variable)
		if l != r {
			s := genUniqueVar()
			bds.put(l, s)
			bds.put(r, s)
			// Otherwise already matche
		}
	} else {
		// lV <= R
		bds.put(l, R)
	}
	return true
}

func (v variable) unify(bds *Bindings) Term {
	t := bds.unifyVar(v)
	if t.Type() != ttVar {
		return t.unify(bds)
	}

	return t
}

func (v variable) export(bds *Bindings) Term {
	t := bds.unifyVar(v)
	if t.Type() != ttVar {
		return t.export(bds)
	}

	vl := t.(variable)
	if vl.isR() {
		s := genUniqueVar()
		bds.put(vl, s)
		return s
	}
	return t
}

var gUniqueVarChan chan variable

func init() {
	gUniqueVarChan = make(chan variable, 10)
	go func() {
		counter := 0
		for {
			gUniqueVarChan <- gV(counter)
			counter ++
		}
	}()
}

func genUniqueVar() variable {
	return <-gUniqueVarChan
}

/* Complex term: *ComplexTerm */

type ComplexTerm struct {
	Functor atom
	Args    []Term
}

// CT creates a *ComplexTerm instance with functor and args.
// The arg with type Term will be inserted directly, and an string will be
// converted into either an Atom or an Variable using TermFromString.
//
// NOTE: using V("v"), you can create variables with lower case prefix. This is
// legal in go-prolog.
func CT(functor atom, args ...interface{}) *ComplexTerm {
	return &ComplexTerm{Functor: functor, Args: L(args...)}
}

func (ct *ComplexTerm) Type() int {
	return ttComplex
}

func (ct *ComplexTerm) replaceVars(bds VarBindings) Term {
	newCt := &ComplexTerm{Functor: ct.Functor, Args: make([]Term, len(ct.Args))}
	for i, arg := range ct.Args {
		newCt.Args[i] = arg.replaceVars(bds)
	}
	return newCt
}

func (ct *ComplexTerm) Key() int {
	//return fmt.Sprintf("%s/%d", ct.Functor, len(ct.Args))
	return int(ct.Functor)*1024 | len(ct.Args)
}

func (ct *ComplexTerm) String() string {
	var buf bytes.Buffer
	buf.Write([]byte(ct.Functor.String()))
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

func (l *ComplexTerm) Match(R Term, bds *Bindings) bool {
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

func (ct *ComplexTerm) unify(bds *Bindings) Term {
	newArgs := make([]Term, len(ct.Args))
	for i, arg := range ct.Args {
		newArgs[i] = arg.unify(bds)
	}
	return &ComplexTerm{Functor: ct.Functor, Args: newArgs}
}

func (ct *ComplexTerm) export(bds *Bindings) Term {
	newArgs := make([]Term, len(ct.Args))
	for i, arg := range ct.Args {
		newArgs[i] = arg.export(bds)
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

func (l List) replaceVars(bds VarBindings) Term {
	newL := make(List, len(l))
	for i, el := range l {
		newL[i] = el.replaceVars(bds)
	}
	return newL
}

func (l List) Match(R Term, bds *Bindings) bool {
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

func (l List) unify(bds *Bindings) Term {
	newL := make(List, len(l))
	for i, el := range l {
		newL[i] = el.unify(bds)
	}
	return newL
}

func (l List) export(bds *Bindings) Term {
	newL := make(List, len(l))
	for i, el := range l {
		newL[i] = el.export(bds)
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

func (l HeadTail) replaceVars(bds VarBindings) Term {
	return HeadTail{Head: l.Head.replaceVars(bds),
		Tail: l.Tail.replaceVars(bds)}
}

func (l HeadTail) Match(R Term, bds *Bindings) bool {
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

func (l HeadTail) unify(bds *Bindings) Term {
	head := l.Head.unify(bds)
	tail := l.Tail.unify(bds)
	if tl, ok := tail.(List); ok {
		// merge back to List
		return append(List{head}, tl...)
	}
	return HeadTail{Head: head, Tail: tail}
}

func (l HeadTail) export(bds *Bindings) Term {
	head := l.Head.export(bds)
	tail := l.Tail.export(bds)
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

func (at FirstLeft) replaceVars(bds VarBindings) Term {
	return FirstLeft{First: at.First.replaceVars(bds),
		Left: at.Left.replaceVars(bds)}
}

func (l FirstLeft) Match(R Term, bds *Bindings) bool {
	if R.Type() != ttAtom {
		return false
	}

	switch r := R.(type) {
	case atom:
		rr := r.String()
		if len(rr) < 1 {
			return false
		}

		if !matchTerm(l.First, A(rr[0:1]), bds) {
			return false
		}
		if !matchTerm(l.Left, A(rr[1:]), bds) {
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

func (at FirstLeft) unify(bds *Bindings) Term {
	first := at.First.unify(bds)
	left := at.Left.unify(bds)

	if fst, ok := first.(atom); ok {
		if lft, ok := left.(atom); ok {
			return A(fst.String() + lft.String())
		}
	}
	return FirstLeft{First: first, Left: left}
}

func (at FirstLeft) export(bds *Bindings) Term {
	first := at.First.export(bds)
	left := at.Left.export(bds)

	if fst, ok := first.(atom); ok {
		if lft, ok := left.(atom); ok {
			return A(fst.String() + lft.String())
		}
	}
	return FirstLeft{First: first, Left: left}
}

const (
	opGt = iota // >
	opGe        // >=
	opLt        // <
	opLe        // <=
	opNe        // !=

	opIs // is

	opPlus  // +
	opMinus // -
	opMul   // *
	opDiv   // /
)

var OpNames map[int]string = map[int]string{
	opGt: ">",
	opGe: ">=",
	opLt: "<",
	opLe: "<=",
	opNe: "!=",

	opIs: "is",

	opPlus:  "+",
	opMinus: "-",
	opMul:   "*",
	opDiv:   "/",
}

/* buildin/2: *buildin2 */

type buildin2 struct {
	Op   int
	L, R Term
}

func Op(l, op, r interface{}) *buildin2 {
	res := buildin2{L: term(l), R: term(r)}
	switch op.(string) {
	case ">":
		res.Op = opGt

	case ">=", "=>":
		res.Op = opGe

	case "<":
		res.Op = opLt

	case "<=", "=<":
		res.Op = opLe

	case "=\\=", "!=":
		res.Op = opNe

	case "+":
		res.Op = opPlus
	case "-":
		res.Op = opMinus
	case "*":
		res.Op = opMul
	case "/":
		res.Op = opDiv

	case "is":
		res.Op = opIs

	default:
		panic(fmt.Sprintf("Unknown op-string: %s", op))
	}

	return &res
}

func Is(l, r interface{}) *buildin2 {
	return Op(l, "is", r)
}

func (bi *buildin2) String() string {
	return fmt.Sprintf("%v %v %v", bi.L, OpNames[bi.Op], bi.R)
}

func (bi *buildin2) Type() int {
	return ttBuildin
}

// replace query variabls
func (bi *buildin2) replaceVars(bds VarBindings) (newT Term) {
	return &buildin2{Op: bi.Op,
		L: bi.L.replaceVars(bds), R: bi.R.replaceVars(bds)}
}

// l: the receiver
// l and R has be unifyVar before called
// if l is not Variable, R is not Variable
func (l *buildin2) Match(R Term, bds *Bindings) bool {
	if r, ok := R.(*buildin2); ok {
		if l.Op != r.Op {
			return false
		}
		return matchTerm(l.L, r.L, bds) && matchTerm(l.R, r.R, bds)
	}

	return false
}

func (bi *buildin2) unify(bds *Bindings) Term {
	return &buildin2{Op: bi.Op,
		L: bi.L.unify(bds), R: bi.R.unify(bds)}
}

func (bi *buildin2) export(bds *Bindings) Term {
	return &buildin2{Op: bi.Op,
		L: bi.L.export(bds), R: bi.R.export(bds)}
}

func isNumber(T Term) bool {
	return T.Type() == ttInt
}

func (bi *buildin2) compute() Term {
	L := bi.L
	if !isNumber(L) {
		L = computeTerm(L)
		if L == nil {
			return nil
		}
	}

	R := bi.R
	if !isNumber(R) {
		R = computeTerm(R)
		if R == nil {
			return nil
		}
	}

	if L.Type() == ttInt && R.Type() == ttInt {
		l, r := L.(Integer), R.(Integer)
		var res Integer
		switch bi.Op {
		case opPlus:
			res = l + r
		case opMinus:
			res = l - r
		case opMul:
			res = l * r
		case opDiv:
			res = l / r
		default:
			return nil
		}

		return res
	}

	return nil
}

/* pVarBindings: gV/rV -> pV */
type pVarBindings struct {
	rList []*variable
	gMap map[variable]variable
	Count int
}

func (bds *pVarBindings) String() string {
	var buf bytes.Buffer
	buf.WriteRune('[')
	first := true
	if bds != nil {
		for i, vl := range bds.rList {
			if vl == nil {
				continue
			}
			if first {
				first = false
			} else {
				buf.WriteRune(' ')
			}
			
			buf.WriteString(fmt.Sprintf("%v->%v", rV(i), vl))
		}
		
		keys := make([]variable, 0, len(bds.gMap))
		for v := range bds.gMap {
			keys = append(keys, v)
		}
		villa.SortF(len(keys), func(i, j int) bool {
			return keys[i].String() < keys[j].String()
		}, func(i, j int) {
			keys[i], keys[j] = keys[j], keys[i]
		})
		
		for _, v := range(keys) {
			vl := bds.gMap[v]
			
			if first {
				first = false
			} else {
				buf.WriteRune(' ')
			}
			
			buf.WriteString(fmt.Sprintf("%v->%v", v, vl))
		}
		
	}
	buf.WriteRune(']')
	return buf.String()
}

func (bds *pVarBindings) get(v variable) (newV variable) {
	if v.isR() {
		pv := bds.rList[v.rIndex()]
		if pv != nil {
			return *pv
		}
		
		newV = pV(bds.Count)
		bds.rList[v.rIndex()] = &newV
		bds.Count ++
		return newV
	}
	
	newV, ok := bds.gMap[v]
	if ok {
		return newV
	}
	newV = pV(bds.Count)
	
	if bds.gMap == nil {
		bds.gMap = make(map[variable]variable)
	}
	bds.gMap[v] = newV
	bds.Count ++

	return newV
}

func (bds *pVarBindings) each(callback func(v, vl variable)) {
	for i, vl := range bds.rList {
		if vl != nil {
			callback(rV(i), *vl)
		}
	}
	
	for v, vl := range bds.gMap {
		callback(v, vl)
	}
}

func newPVarBindings(nRVars int) *pVarBindings {
	return &pVarBindings{rList: make([]*variable, nRVars)}
}

/* rVarBindings: gV -> rV */
type rVarBindings map[variable]variable

func (bds rVarBindings) get(v variable) variable {
	newV, ok := bds[v]
	if ok {
		return newV
	}
	newV = rV(len(bds))
	bds[v] = newV

	return newV
}

/* Bindings: Variable map to its value as a Term */

type Bindings struct {
	rList []Term
	gMap map[variable]Term
}

func newBindings(nRVars int) *Bindings {
	return &Bindings{rList: make([]Term, nRVars)}
}

func newBindingsFrom(bds *Bindings) *Bindings {
	return &Bindings{rList: make([]Term, len(bds.rList))}
}

func (bds *Bindings) String() string {
	var buf bytes.Buffer
	buf.WriteRune('[')
	first := true
	if bds != nil {
		for i, vl := range bds.rList {
			if vl == nil {
				continue
			}
			if first {
				first = false
			} else {
				buf.WriteRune(' ')
			}
			
			buf.WriteString(fmt.Sprintf("%v->%v", rV(i), vl))
		}
		
		keys := make([]variable, 0, len(bds.gMap))
		for v := range bds.gMap {
			keys = append(keys, v)
		}
		villa.SortF(len(keys), func(i, j int) bool {
			return keys[i].String() < keys[j].String()
		}, func(i, j int) {
			keys[i], keys[j] = keys[j], keys[i]
		})
		
		for _, v := range(keys) {
			vl := bds.gMap[v]
			
			if first {
				first = false
			} else {
				buf.WriteRune(' ')
			}
			
			buf.WriteString(fmt.Sprintf("%v->%v", v, vl))
		}
		
	}
	buf.WriteRune(']')
	return buf.String()
}

func (bds *Bindings) put(v variable, t Term) {
	if v.isR() {
		bds.rList[v.rIndex()] = t
		return
	}
	
	bds.putG(v, t)
}

func (bds *Bindings) putG(v variable, t Term) {
	if bds.gMap == nil {
		bds.gMap = make(map[variable]Term)
	}
	bds.gMap[v] = t
}

// returns nil if no bindings
func (bds *Bindings) get(v variable) Term {
	if (bds == nil) {
		return nil
	}
	
	if v.isR() {
		return bds.rList[v.rIndex()]
	}
	
	return bds.gMap[v]
}

func (bds *Bindings) RVarCount() int {
	return len(bds.rList)
}

// keep unify until t is no longer a Variable, but no further unify
func (bds *Bindings) unifyVar(t Term) Term {
	for t.Type() == ttVar {
		v := t.(variable)
		i := bds.get(v)
		if i == nil {
			break
		}
		t = i
	}
	return t
}

// c = a + b...
func (a *Bindings) combine(b *Bindings) (c *Bindings) {
	if b == nil {
		return a
	}
	
	if a != nil {
		for i, v := range a.rList {
			if v != nil {
				b.rList[i] = v
			}
		}
		for v, vl := range a.gMap {
			b.putG(v, vl)
		}
	}
	return b
}

/* matchTerm */

func matchTerm(L, R Term, bds *Bindings) (succ bool) {
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

/* computeTerm */

func computeTerm(T Term) Term {
	switch T.Type() {
	case ttInt:
		return T
	case ttBuildin:
		t := T.(*buildin2)
		return t.compute()
	}
	return nil
}
