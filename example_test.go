package gocalc_test

import (
	"context"
	"errors"
	"fmt"

	gocalc "github.com/rahuldr07/go-calc"
)

func ExampleEval() {
	r, err := gocalc.Eval(context.Background(), "0.1 + 0.2")
	if err != nil {
		panic(err)
	}
	fmt.Println(r.Text, r.Status, r.Exact)
	fmt.Println("float64 would say", r.Float64)
	// Output:
	// 0.3 exact 3/10
	// float64 would say 0.30000000000000004
}

func ExampleEval_proven() {
	r, _ := gocalc.Eval(context.Background(), "sqrt(2)", gocalc.Digits(30))
	fmt.Println(r.Text, r.Status)
	// Output:
	// 1.41421356237309504880168872421 proven
}

func ExampleEval_refusal() {
	_, err := gocalc.Eval(context.Background(), "1 m + 2 s")
	var refusal *gocalc.Refusal
	if errors.As(err, &refusal) {
		fmt.Println(refusal.Reason)
	}
	// Output:
	// cannot add length (m) and time (s)
}

func ExampleEvalSheet() {
	sheet, _ := gocalc.EvalSheet(context.Background(), []string{
		"distance = 412 km",
		"fuel = 6.8 L/100km * distance in L",
		"fuel * 1.92 in L",
	})
	for _, r := range sheet.Results {
		fmt.Printf("%-36s %-12s %s\n", r.Source, r.Text, r.Status)
	}
	// Output:
	// distance = 412 km                    412 km       exact
	// fuel = 6.8 L/100km * distance in L   28.016 L     exact
	// fuel * 1.92 in L                     53.79072 L   exact
}
