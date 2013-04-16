package prolog

import (
	"fmt"
	"testing"
)

func ctFunc(name Atom) func(args ...interface{}) *ComplexTerm {
	return func(args ...interface{}) *ComplexTerm {
		return CT(name, args...)
	}
}

func match(m *Machine, ct *ComplexTerm) int {
	slns := make(chan Bindings)
	go m.Match(ct, slns)
	fmt.Println("Match fact", ct, ": ")
	count := 0
	for sln := range slns {
		count ++
		fmt.Println("    For", sln)
	}
	if count > 0 {
		fmt.Println()
	} else {
		fmt.Println("    false")
	}
	
	return count
}

func assertCount(t *testing.T, exp, act int) {
	if exp != act {
		t.Errorf("Expected %d solutions, but got %d solutions.", exp, act)
	}
}

const(
	B = "B"
	C = "C"
	D = "D"
	P = "P"
	Q = "Q"
	W = "W"
	X = "X"
	Y = "Y"
	Z = "Z"
	X1 = "X1"
	Y1 = "Y1"
	X2 = "X2"
	Y2 = "Y2"
)
	
func TestFact(t *testing.T) {
	m := NewMachine()
	
	line := ctFunc("line")
	point := ctFunc("point")
	vertical := ctFunc("vertical")
	horizontal := ctFunc("horizontal")
	same := ctFunc("same")
	like := ctFunc("like")

	m.AddFact(vertical(line(point(X, Y), point(X, Z))))
	m.AddFact(horizontal(line(point(X, Y), point(Z, Y))))

	m.AddFact(same(X, X, X))

	m.AddFact(like("david", "food"))
	m.AddFact(like("david", "money"))
	m.AddFact(like("xmz", "money"))
	m.AddFact(like("xmz", "house"))

	assertCount(t, 1, match(m,
		vertical(line(point(1, 2), point(1, 3)))))

	assertCount(t, 0, match(m,
		vertical(line(point(1, 2), point("1", 3)))))

	assertCount(t, 0, match(m,
		vertical(line(point("1", "2"), point("5", "3")))))

	assertCount(t, 1, match(m,
		vertical(line(point("1", "2"), point(Q, "3")))))

	assertCount(t, 1, match(m,
		vertical(line(point("1", "2"), P))))

	assertCount(t, 1, match(m,
		vertical(line(P, point("1", "2")))))

	assertCount(t, 1, match(m,
		vertical(line(point("1", Y1), point("X2", "Y2")))))

	assertCount(t, 1, match(m,
		vertical(line(point(X1, "1"), point(X2, Y2)))))

	assertCount(t, 1, match(m, same(B, C, D)))
	assertCount(t, 1, match(m, same("a", C, D)))
	
	assertCount(t, 2, match(m, like("david", "What")))
	assertCount(t, 2, match(m, like("Who", "money")))
	assertCount(t, 4, match(m, like(X, Y)))

	fmt.Printf("Machine: %+v\n", m)
}

func TestRule_Simple(t *testing.T) {
	m := NewMachine()

	f := ctFunc("f")
	g := ctFunc("g")
	h := ctFunc("h")
	all := ctFunc("all")
	
	m.AddFact(f("a"))
	m.AddFact(f("b"))

	m.AddFact(g("a"))
	m.AddFact(g("b"))

	m.AddFact(h("b"))

	m.AddRule(R(all(X),
		f(X),
		g(X),
		h(X)))

	assertCount(t, 1, match(m, all(X)))

	fmt.Printf("Machine: %+v\n", m)
}

func TestRule2(t *testing.T) {
	m := NewMachine()
	
	parent := ctFunc("parent")
	descendant := ctFunc("descendant")

	m.AddFact(parent("david", "xiaoxi"))
	m.AddFact(parent("laotaiye", "david"))
	m.AddFact(parent("laolaotaiye", "laotaiye"))

	m.AddRule(R(descendant(X, Y), parent(X, Y)))

	m.AddRule(R(descendant(X, Y),
		parent(X, Z),
		descendant(Z, Y)))

	assertCount(t, 3, match(m, parent(X, Y)))
	assertCount(t, 6, match(m, descendant(P, Q)))

	fmt.Printf("Machine: %+v\n", m)
}

func TestProgram_Rev(t *testing.T) {
	reverse := ctFunc("reverse")
	
	m := NewMachine()
	
	// reverse([], X, X).
	m.AddFact(reverse(L(), X, X))
	// reverse([X|Y], Z, W) :-
	//     reverse(Y, [X|Z], W).
	m.AddRule(R(reverse(HT(X, Y), Z, W),
		reverse(Y, HT(X, Z), W)))
	
	assertCount(t, 1, match(m, reverse(L(), L(), X)))
	assertCount(t, 1, match(m, reverse(L("1", L("2"), "3"), L(), X)))

	fmt.Printf("Machine: %+v\n", m)
}

func TestFirstLeft(t *testing.T) {
	reverse := ctFunc("reverse")
	
	m := NewMachine()
	m.AddFact(reverse("", X, X))
	m.AddRule(R(reverse(FL(X, Y), Z, W),
		reverse(Y, FL(X, Z), W)))
		
	assertCount(t, 1, match(m, reverse("", "", X)))
	assertCount(t, 1, match(m, reverse("abc", "", X)))

	fmt.Printf("Machine: %+v\n", m)
}
