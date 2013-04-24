package main

import (
	"github.com/daviddengcn/go-prolog"
	"fmt"
	"os"
	"runtime/pprof"
	"time"
	"log"
)

const (
	A  = "A"
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
	Z1 = "Z1"
	Z2 = "Z2"
)

func ctFunc(name string) func(args ...interface{}) *plg.ComplexTerm {
	return func(args ...interface{}) *plg.ComplexTerm {
		return plg.CT(plg.A(name), args...)
	}
}

func calcInt(m *plg.Machine, ct *plg.ComplexTerm, rV string) (vl []int) {
	slns := m.Match(ct)
	fmt.Println("Match fact", ct, ": ")
	count := 0
	if slns != nil {
		for sln := range slns {
			count++
			fmt.Println("    For", sln)
			vl = append(vl, int(sln.Get(plg.V(rV)).(plg.Integer)))
		}
	}
	if count > 0 {
		fmt.Println()
	} else {
		fmt.Println("    false")
	}

	return vl
}

func grid(w, h int) {
	grid := ctFunc("grid")

	m := plg.NewMachine()

	m.AddFact(grid(X, 0, 1))
	m.AddFact(grid(0, X, 1))
	m.AddRule(plg.R(grid(X, Y, Z),
		plg.Op(X, ">", 0),
		plg.Op(Y, ">", 0),
		plg.Is(X1, plg.Op(X, "-", 1)),
		grid(X1, Y, Z1),
		plg.Is(Y1, plg.Op(Y, "-", 1)),
		grid(X, Y1, Z2),
		plg.Is(Z, plg.Op(Z1, "+", Z2))))

	//	calcInt(m, grid(1, 1, X), V(X))
	// calcInt(m, grid(2, 2, X), V(X))

	calcInt(m, grid(w, h, X), X)

	fmt.Printf("Machine: %+v\n", m)
}

func main() {
	f, err := os.Create("simple.cpu.prof")
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	defer func() {
		pprof.StopCPUProfile()
	}()
	
	fmt.Printf("%#v\n", pprof.Profiles()[0])
	fmt.Printf("%#v\n", pprof.Profiles()[1])
	fmt.Printf("%#v\n", pprof.Profiles()[2])

	start := time.Now()
	grid(11, 11)
	end := time.Now()
	dur := end.Sub(start)
	fmt.Println(dur)
}
