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
	slns := m.Match(ct)
	fmt.Println("Match fact", ct, ": ")
	count := 0
	if slns != nil {
		for sln := range slns {
			count++
			fmt.Println("    For", sln)
		}
	}
	if count > 0 {
		fmt.Println()
	} else {
		fmt.Println("    false")
	}

	return count
}

func calcInt(m *Machine, ct *ComplexTerm, rV Variable) (vl []int) {
	slns := m.Match(ct)
	fmt.Println("Match fact", ct, ": ")
	count := 0
	if slns != nil {
		for sln := range slns {
			count++
			fmt.Println("    For", sln)
			vl = append(vl, int(sln[rV].(Integer)))
		}
	}
	if count > 0 {
		fmt.Println()
	} else {
		fmt.Println("    false")
	}

	return vl
}

func assertCount(t *testing.T, exp, act int) {
	if exp != act {
		t.Errorf("Expected %d solutions, but got %d solutions.", exp, act)
	}
}

const (
	B  = "B"
	C  = "C"
	D  = "D"
	F  = "F"
	N  = "N"
	P  = "P"
	Q  = "Q"
	W  = "W"
	X  = "X"
	Y  = "Y"
	Z  = "Z"
	F1 = "F1"
	F2 = "F2"
	N1 = "N1"
	N2 = "N2"
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

func TestProgram_Factorial(t *testing.T) {
	factorial := ctFunc("factorial")

	m := NewMachine()

	m.AddFact(factorial(0, 1))
	m.AddRule(R(factorial(N, F),
			Op(N, ">", 0),
			Is(N1, Op(N, "-", 1)),
			factorial(N1, F1),
			Is(F, Op(N, "*", F1))))
	calcInt(m, factorial(0, X), X)
	calcInt(m, factorial(5, X), X)
	fmt.Printf("Machine: %+v\n", m)
}

func TestProgram_Fibonacci(t *testing.T) {
	fibonacci := ctFunc("fibonacci")

	m := NewMachine()

	m.AddFact(fibonacci(1, 1))
	m.AddFact(fibonacci(2, 1))
	m.AddRule(R(fibonacci(N, F),
			Op(N, ">", 2),
			Is(N1, Op(N, "-", 1)),
			fibonacci(N1, F1),
			Is(N2, Op(N, "-", 2)),
			fibonacci(N2, F2),
			Is(F, Op(F1, "+", "F2"))))
	calcInt(m, fibonacci(1, X), X)
	calcInt(m, fibonacci(2, X), X)
	calcInt(m, fibonacci(3, X), X)
	calcInt(m, fibonacci(4, X), X)
	calcInt(m, fibonacci(5, X), X)
	calcInt(m, fibonacci(6, X), X)
	calcInt(m, fibonacci(7, X), X)
	fmt.Printf("Machine: %+v\n", m)
}

/*
func TestProgram_Grid(t *testing.T) {
	grid := ctFunc("grid")

	m := NewMachine()

	m.AddFact(grid(X, 1, 1))
	m.AddFact(grid(1, X, 1))
	m.AddRule(R(grid(X, Y, Z),
		Or(X > 1, Y > 1),
//		X1 is X - 1, 
//		grid(X1, Y, Z1),
//		Y1 is Y - 1,
//		grid(X, Y1, Z2),
//		Z is Z1 + Z2))
	calcInt(m, grid(1, 1, X), X)

	fmt.Printf("Machine: %+v\n", m)
}
*/
