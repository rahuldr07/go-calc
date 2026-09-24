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

func TestCommaGroupsOnlyJoinWhenWellFormed(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{"1,000", "1,000"},
		{"18,50,000", "18,50,000"},
		{"1,00,00,000", "1,00,00,000"},
		{"1,234,567", "1,234,567"},
		{"1,000.5", "1,000.5"},
		{"48,18", "48"},
		{"1,2", "1"},
		{"12,345,67", "12,345"},
		{"1,0000", "1"},
		{"1,", "1"},
		{"1234,567", "1234"},
		{"12345,678", "12345"},
		{"1e5,000", "1e5"},
		{"1.5,000", "1.5"},
		{"1_000,000", "1_000,000"},
		{"1_0,000", "1_0,000"},
		{"1_0000,000", "1_0000"},
	}
	for _, c := range cases {
		toks, err := Scan(c.src)
		if err != nil {
			t.Fatalf("Scan(%q): %v", c.src, err)
		}
		if toks[0].Kind != Num {
			t.Errorf("Scan(%q) first token kind = %v, want Num", c.src, toks[0].Kind)
			continue
		}
		if toks[0].Text != c.want {
			t.Errorf("Scan(%q) first number = %q, want %q", c.src, toks[0].Text, c.want)
		}
	}
}

func TestMalformedGroupingLeavesACommaToken(t *testing.T) {
	toks, err := Scan("48,18")
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(toks) != 4 || toks[1].Kind != Comma || toks[2].Text != "18" {
		t.Fatalf("Scan(48,18) = %v, want 48 , 18 and EOF", toks)
	}
}
