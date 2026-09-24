package lex

import "testing"

func TestScanNumbersAndOperators(t *testing.T) {
	toks, err := Scan("12.3 + 1_000 * 0xFF")
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	want := []struct {
		kind Kind
		text string
	}{
		{Num, "12.3"},
		{Op, "+"},
		{Num, "1_000"},
		{Op, "*"},
		{Num, "0xFF"},
		{EOF, ""},
	}
	if len(toks) != len(want) {
		t.Fatalf("got %d tokens, want %d: %v", len(toks), len(want), toks)
	}
	for i, w := range want {
		if toks[i].Kind != w.kind || toks[i].Text != w.text {
			t.Errorf("token %d = (%v, %q), want (%v, %q)", i, toks[i].Kind, toks[i].Text, w.kind, w.text)
		}
	}
}

func TestScanPositions(t *testing.T) {
	toks, err := Scan("a =\n  42")
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if toks[2].Pos.Line != 2 || toks[2].Pos.Col != 3 {
		t.Errorf("42 at %v, want line 2 col 3", toks[2].Pos)
	}
}

func TestScanIndianGroupingAndComment(t *testing.T) {
	toks, err := Scan("18,50,000 # salary")
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if toks[0].Kind != Num || toks[0].Text != "18,50,000" {
		t.Errorf("got (%v, %q), want grouped number", toks[0].Kind, toks[0].Text)
	}
	if toks[1].Kind != Comment {
		t.Errorf("got %v, want Comment", toks[1].Kind)
	}
}

func TestScanUnknownRune(t *testing.T) {
	if _, err := Scan("2 $ 3"); err == nil {
		t.Fatal("want error for unknown rune")
	}
}

func FuzzScan(f *testing.F) {
	f.Add("12.3 + 1_000")
	f.Add("18,50,000 # c")
	f.Add("0xFF ± 2e-3")
	f.Fuzz(func(t *testing.T, src string) {
		toks, err := Scan(src)
		if err != nil {
			return
		}
		if len(toks) == 0 || toks[len(toks)-1].Kind != EOF {
			t.Fatalf("scan of %q did not end in EOF", src)
		}
	})
}
