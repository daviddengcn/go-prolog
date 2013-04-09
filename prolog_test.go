package prolog

import (
	"fmt"
	"testing"
)

func matchFact(m *Machine, ct *ComplexTerm) {
	slns := make(chan Solution)
	go m.MatchFact(ct, slns)
	fmt.Println("Match fact ", ct, ": ")
	found := false
	for s := range slns {
		if !found {
			found = true
			fmt.Println("    true")
		}
		fmt.Printf("    %s\n", s[S_FACT])
		fmt.Print("        ")
		for k, v := range s {
			if k != S_FACT {
				fmt.Print(" ", k, " = ", v)
			}
		}
		fmt.Println()
	}
	if found {
		fmt.Println()
	} else {
		fmt.Println("false")
	}
}

func TestProlog(t *testing.T) {
	m := NewMachine()

	m.AddFact(NewComplexTerm("vertical",
		NewComplexTerm("line",
			NewComplexTerm("point", V("X"), V("Y")),
			NewComplexTerm("point", V("X"), V("Z")))))

	m.AddFact(NewComplexTerm("horizontal",
		NewComplexTerm("line",
			NewComplexTerm("point", V("X"), V("Y")),
			NewComplexTerm("point", V("Z"), V("Y")))))

	m.AddFact(NewComplexTerm("same", V("X"), V("X"), V("X")))

	m.AddFact(NewComplexTerm("like", A("david"), A("food")))
	m.AddFact(NewComplexTerm("like", A("david"), A("money")))
	m.AddFact(NewComplexTerm("like", A("xmz"), A("money")))
	m.AddFact(NewComplexTerm("like", A("xmz"), A("house")))

	matchFact(m, NewComplexTerm("vertical",
		NewComplexTerm("line",
			NewComplexTerm("point", A("1"), A("2")),
			NewComplexTerm("point", A("1"), A("3")))))

	matchFact(m, NewComplexTerm("vertical",
		NewComplexTerm("line",
			NewComplexTerm("point", A("1"), A("2")),
			NewComplexTerm("point", A("5"), A("3")))))

	matchFact(m, NewComplexTerm("vertical",
		NewComplexTerm("line",
			NewComplexTerm("point", A("1"), A("2")),
			NewComplexTerm("point", V("Q"), A("3")))))

	matchFact(m, NewComplexTerm("vertical",
		NewComplexTerm("line",
			NewComplexTerm("point", A("1"), A("2")),
			V("P"))))

	matchFact(m, NewComplexTerm("vertical",
		NewComplexTerm("line",
			V("P"),
			NewComplexTerm("point", A("1"), A("2")))))

	matchFact(m, NewComplexTerm("vertical",
		NewComplexTerm("line",
			NewComplexTerm("point", A("1"), V("Y1")),
			NewComplexTerm("point", V("X2"), V("Y2")))))

	matchFact(m, NewComplexTerm("vertical",
		NewComplexTerm("line",
			NewComplexTerm("point", V("X1"), A("1")),
			NewComplexTerm("point", V("X2"), V("Y2")))))

	matchFact(m, NewComplexTerm("same", V("A"), V("B"), V("C")))
	matchFact(m, NewComplexTerm("same", A("a"), V("B"), V("C")))
	matchFact(m, NewComplexTerm("like", A("david"), V("What")))
	matchFact(m, NewComplexTerm("like", V("Who"), A("money")))

	matchFact(m, NewComplexTerm("like", V("X"), V("Y")))

	fmt.Println("Machine:", m)
}
