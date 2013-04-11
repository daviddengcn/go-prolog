package prolog

import (
	"fmt"
	"testing"
)

func match(m *Machine, ct *ComplexTerm) {
	slns := make(chan Context)
	go m.Match(ct, slns)
	fmt.Println("Match fact", ct, ": ")
	found := false
	for sln := range slns {
		found = true
		fmt.Println("    For", sln)
	}
	if found {
		fmt.Println()
	} else {
		fmt.Println("    false")
	}
}

func TestFact(t *testing.T) {
	m := NewMachine()

	m.AddFact(CT("vertical",
		CT("line",
			CT("point", V("X"), V("Y")),
			CT("point", V("X"), V("Z")))))

	m.AddFact(CT("horizontal",
		CT("line",
			CT("point", V("X"), V("Y")),
			CT("point", V("Z"), V("Y")))))

	m.AddFact(CT("same", V("X"), V("X"), V("X")))

	m.AddFact(CT("like", A("david"), A("food")))
	m.AddFact(CT("like", A("david"), A("money")))
	m.AddFact(CT("like", A("xmz"), A("money")))
	m.AddFact(CT("like", A("xmz"), A("house")))

	match(m, CT("vertical",
		CT("line",
			CT("point", A("1"), A("2")),
			CT("point", A("1"), A("3")))))

	match(m, CT("vertical",
		CT("line",
			CT("point", A("1"), A("2")),
			CT("point", A("5"), A("3")))))

	match(m, CT("vertical",
		CT("line",
			CT("point", A("1"), A("2")),
			CT("point", V("Q"), A("3")))))

	match(m, CT("vertical",
		CT("line",
			CT("point", A("1"), A("2")),
			V("P"))))

	match(m, CT("vertical",
		CT("line",
			V("P"),
			CT("point", A("1"), A("2")))))

	match(m, CT("vertical",
		CT("line",
			CT("point", A("1"), V("Y1")),
			CT("point", V("X2"), V("Y2")))))

	match(m, CT("vertical",
		CT("line",
			CT("point", V("X1"), A("1")),
			CT("point", V("X2"), V("Y2")))))

	match(m, CT("same", V("A"), V("B"), V("C")))
	match(m, CT("same", A("a"), V("B"), V("C")))
	match(m, CT("like", A("david"), V("What")))
	match(m, CT("like", V("Who"), A("money")))

	match(m, CT("like", V("X"), V("Y")))

	fmt.Printf("Machine: %+v\n", m)
}

func TestRule(t *testing.T) {
	m := NewMachine()

	m.AddFact(CT("f", A("a")))
	m.AddFact(CT("f", A("b")))

	m.AddFact(CT("g", A("a")))
	m.AddFact(CT("g", A("b")))

	m.AddFact(CT("h", A("b")))

	m.AddRule(R(CT("all", V("X")),
		CT("f", V("X")),
		CT("g", V("X")),
		CT("h", V("X"))))

	match(m, CT("all", V("Y")))

	fmt.Printf("Machine: %+v\n", m)
}
