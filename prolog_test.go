package plg

import (
	"fmt"
	"testing"
)

func ctFunc(name string) func(args ...interface{}) *ComplexTerm {
	return func(args ...interface{}) *ComplexTerm {
		return CT(A(name), args...)
	}
}

func match(m *Machine, ct *ComplexTerm) int {
	slns := m.Match(ct)
	fmt.Println("Match fact", ct, ": ")
	count := 0
	if slns != nil {
		for sln := range slns {
			count++
			fmt.Println("    For", sln, ", i.e.", ct.unify(sln))
		}
	}
	if count > 0 {
		fmt.Println()
	} else {
		fmt.Println("    false")
	}

	return count
}

func calcInt(m *Machine, ct *ComplexTerm, rV variable) (vl []int) {
	slns := m.Match(ct)
	fmt.Println("Match fact", ct, ": ")
	count := 0
	if slns != nil {
		for sln := range slns {
			count++
			fmt.Println("    For", sln)
			vl = append(vl, int(sln.get(rV).(Integer)))
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
	Z1 = "Z1"
	Z2 = "Z2"
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
	calcInt(m, factorial(0, X), V(X))
	calcInt(m, factorial(5, X), V(X))
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
	calcInt(m, fibonacci(1, X), V(X))
	calcInt(m, fibonacci(2, X), V(X))
	calcInt(m, fibonacci(3, X), V(X))
	calcInt(m, fibonacci(4, X), V(X))
	calcInt(m, fibonacci(5, X), V(X))
	calcInt(m, fibonacci(6, X), V(X))
	calcInt(m, fibonacci(7, X), V(X))
	fmt.Printf("Machine: %+v\n", m)
}

func TestProgram_Grid(t *testing.T) {
	grid := ctFunc("grid")

	m := NewMachine()

	m.AddFact(grid(X, 0, 1))
	m.AddFact(grid(0, X, 1))
	m.AddRule(R(grid(X, Y, Z),
		Op(X, ">", 0),
		Op(Y, ">", 0),
		Is(X1, Op(X, "-", 1)),
		grid(X1, Y, Z1),
		Is(Y1, Op(Y, "-", 1)),
		grid(X, Y1, Z2),
		Is(Z, Op(Z1, "+", Z2))))

	//	calcInt(m, grid(1, 1, X), V(X))
	// calcInt(m, grid(2, 2, X), V(X))

	calcInt(m, grid(9, 9, X), V(X))

	fmt.Printf("Machine: %+v\n", m)
}

func grid(M, N, Z interface{}, bds map[string]interface{}) chan map[string]interface{} {
	out := make(chan map[string]interface{})
	go func() {
		m, n := bds[M.(string)].(int), bds[N.(string)].(int)
		if m == 0 || n == 0 {
			slns := make(chan map[string]interface{})
			go func(out chan map[string]interface{}) {
				out <- map[string]interface{}{Z.(string): 1}
				close(out)
			}(slns)

			for sln := range slns {
				out <- sln
			}
			close(out)
			return
		}

		slns := make(chan map[string]interface{}, 1)
		slns <- make(map[string]interface{})
		close(slns)
		go func() { // M > 0
			m = bds[M.(string)].(int)
			<-slns

			slns1 := make(chan map[string]interface{}, 1)
			slns1 <- make(map[string]interface{})
			close(slns1)
			go func() { // N > 0
				n = bds[N.(string)].(int)
				<-slns1

				var M1 interface{} = string(genUniqueVar())
				bds[M1.(string)] = bds[M.(string)].(int) - 1
				slns2 := make(chan map[string]interface{}, 1)
				slns2 <- nil
				close(slns2)
				go func() { // M1 is M - 1
					<-slns2

					go func() { // grid(M1, N, Z1)
						var Z1 interface{} = string(genUniqueVar())
						z1 := grid(M1, N, Z1, bds)
						go func() { // N1 is N - 1
							bds[Z1.(string)] = (<-z1)[Z1.(string)]

							var N1 interface{} = string(genUniqueVar())
							bds[N1.(string)] = bds[N.(string)].(int) - 1
							var Z2 interface{} = string(genUniqueVar())
							z2 := grid(M, N1, Z2, bds)
							go func() { // grid(M, N1, Z2)
								bds[Z2.(string)] = (<-z2)[Z2.(string)]
								z := bds[Z1.(string)].(int) + bds[Z2.(string)].(int)

								go func() { // Z is Z1 + Z2
									out <- map[string]interface{}{Z.(string): z}
									close(out)
								}()
							}()
						}()
					}()
				}()
			}()
		}()
	}()
	return out
}

func TestProgram_GoGrid(t *testing.T) {
	//	fmt.Println(grid(1, 1));
	//	fmt.Println(grid(2, 2));
	//	fmt.Println(grid(9, 9));

	//	fmt.Println(<-grid("m", "n", "z", map[string]interface{}{"m": 1, "n": 1}))
	//	fmt.Println(<-grid("m", "n", "z", map[string]interface{}{"m": 2, "n": 2}))
	//fmt.Println(<-grid("m", "n", "z", map[string]interface{}{"m": 9, "n": 9}))
	//	fmt.Println(grid(2, 2));
	//	fmt.Println(grid(9, 9));
}

func TestReplaceVars(t *testing.T) {
	f := ctFunc("f")
	a := f(X, Y)
	fmt.Println(a)
	pBds := newPVarBindings(0)
	b := a.replaceVars(pBds)
	fmt.Println(b, pBds)

	assertCount(t, 2, pBds.Count)

	rBds := make(rVarBindings)
	c := a.replaceVars(rBds)
	fmt.Println(c, rBds)
	assertCount(t, 2, len(rBds))

	r := R(f(X, Y), f(X, Z))
	fmt.Println(r)
	rBds = make(rVarBindings)
	r.Head = r.Head.replaceVars(rBds).(*ComplexTerm)
	r.Body = r.Body.replaceGoalVars(rBds)
	fmt.Println(r, rBds)
	assertCount(t, 3, len(rBds))
}
