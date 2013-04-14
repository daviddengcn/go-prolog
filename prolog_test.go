package prolog

import (
	"fmt"
	"testing"
)

func match(m *Machine, ct *ComplexTerm) int {
	slns := make(chan Bindings)
	go m.Match(ct, slns)
	fmt.Println("Match fact", ct, ": ")
	count := 0
	for sln := range slns {
		count++
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

func TestFact(t *testing.T) {
	m := NewMachine()

	m.AddFact(CT("vertical", CT("line",
		CT("point", "X", "Y"), CT("point", "X", "Z"))))

	m.AddFact(CT("horizontal", CT("line",
		CT("point", "X", "Y"), CT("point", "Z", "Y"))))

	m.AddFact(CT("same", "X", "X", "X"))

	m.AddFact(CT("like", "david", "food"))
	m.AddFact(CT("like", "david", "money"))
	m.AddFact(CT("like", "xmz", "money"))
	m.AddFact(CT("like", "xmz", "house"))

	// vertical(line(point(1, 2), point(1, 3)))
	assertCount(t, 1, match(m,
		CT("vertical", CT("line",
			CT("point", "1", "2"),
			CT("point", "1", "3")))))

	// vertical(line(point(1, 2), point(5, 3)))
	assertCount(t, 0, match(m,
		CT("vertical", CT("line",
			CT("point", "1", "2"),
			CT("point", "5", "3")))))

	// vertical(line(point(1, 2), point(Q, 3)))
	assertCount(t, 1, match(m,
		CT("vertical", CT("line",
			CT("point", "1", "2"),
			CT("point", "Q", "3")))))

	// vertical(line(point(1, 2), P))
	assertCount(t, 1, match(m,
		CT("vertical", CT("line",
			CT("point", "1", "2"), "P"))))

	// vertical(line(P, point(1, 2)))
	assertCount(t, 1, match(m,
		CT("vertical", CT("line",
			"P", CT("point", "1", "2")))))

	// vertical(line(point(1, Y1), point(X2, Y2)))
	assertCount(t, 1, match(m,
		CT("vertical", CT("line",
			CT("point", "1", "Y1"),
			CT("point", "X2", "Y2")))))

	// vertical(line(point(X1, 1), point(X2, Y2)))
	assertCount(t, 1, match(m,
		CT("vertical", CT("line",
			CT("point", "X1", "1"),
			CT("point", "X2", "Y2")))))

	// same(A, B, C)
	assertCount(t, 1, match(m, CT("same", "A", "B", "C")))
	// same(a, B, C)
	assertCount(t, 1, match(m, CT("same", "a", "B", "C")))

	// like(david, What)
	assertCount(t, 2, match(m, CT("like", "david", "What")))
	// like(Who, money)
	assertCount(t, 2, match(m, CT("like", "Who", "money")))
	// like(X, Y)
	assertCount(t, 4, match(m, CT("like", "X", "Y")))

	fmt.Printf("Machine: %+v\n", m)
}

func TestRule_Simple(t *testing.T) {
	m := NewMachine()

	m.AddFact(CT("f", "a"))
	m.AddFact(CT("f", "b"))

	m.AddFact(CT("g", "a"))
	m.AddFact(CT("g", "b"))

	m.AddFact(CT("h", "b"))

	m.AddRule(R(CT("all", "X"),
		CT("f", "X"),
		CT("g", "X"),
		CT("h", "X")))

	assertCount(t, 1, match(m, CT("all", "X")))

	fmt.Printf("Machine: %+v\n", m)
}

func TestRule2(t *testing.T) {
	m := NewMachine()

	m.AddFact(CT("parent", "david", "xiaoxi"))
	m.AddFact(CT("parent", "laotaiye", "david"))
	m.AddFact(CT("parent", "laolaotaiye", "laotaiye"))

	m.AddRule(R(CT("descendant", "X", "Y"),
		CT("parent", "X", "Y")))

	m.AddRule(R(CT("descendant", "X", "Y"),
		CT("parent", "X", "Z"),
		CT("descendant", "Z", "Y")))

	assertCount(t, 3, match(m, CT("parent", "X", "Y")))
	assertCount(t, 6, match(m, CT("descendant", "P", "Q")))

	fmt.Printf("Machine: %+v\n", m)
}

func TestProgram_Rev(t *testing.T) {
	const (
		X = "X"
		Y = "Y"
		Z = "Z"
		W = "W"
	)
	reverse := func(args ...interface{}) *ComplexTerm {
		return CT("reverse", args...)
	}

	m := NewMachine()

	// reverse([], X, X).
	m.AddFact(reverse(L(), X, X))
	// reverse([X|Y], Z, W) :-
	//     reverse(Y, [X|Z], W).
	m.AddRule(R(reverse(HT(X, Y), Z, W),
		reverse(Y, HT(X, Z), W)))

	assertCount(t, 1, match(m, reverse(L(), L(), X)))
	assertCount(t, 1, match(m, reverse(L("1", L("2"), "3"), L(), X)))
}
