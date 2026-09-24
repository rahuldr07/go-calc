package doc

import (
	"encoding/json"
	"fmt"
	"testing"
)

func benchLines() []string {
	src := []string{"rate = 3/100", "span = 12", "base = 1250.75"}
	for g := 0; len(src) < 99; g++ {
		src = append(src,
			fmt.Sprintf("a%d = base * (1 + rate) + %d", g, g),
			fmt.Sprintf("b%d = a%d / span", g, g),
			fmt.Sprintf("c%d = sqrt(b%d)", g, g),
			fmt.Sprintf("c%d + a%d", g, g))
	}
	return append(src, "c0 + c23")
}

func benchRequest(lines []string) []byte {
	req, err := json.Marshal(sheetRequest{Lines: lines, Digits: 12})
	if err != nil {
		panic(err)
	}
	return req
}

var benchSink []byte

func checkClean(b *testing.B, lines []Line) {
	b.Helper()
	for _, l := range lines {
		if l.Err != nil {
			b.Fatalf("line %d (%s): %v", l.N, l.Src, l.Err)
		}
	}
}

func BenchmarkHundredLineRecalculation(b *testing.B) {
	src := benchLines()
	if len(src) != 100 {
		b.Fatalf("the document has %d lines, want 100", len(src))
	}
	b.ReportAllocs()
	for b.Loop() {
		checkClean(b, New().SetLines(src))
	}
}

func BenchmarkEditFewDependents(b *testing.B) {
	src := benchLines()
	d := New()
	checkClean(b, d.SetLines(src))
	last := len(src) - 2
	if got := len(d.Edit(last, src[last])); got != 1 {
		b.Fatalf("editing line %d restaled %d lines, want 1", last, got)
	}
	b.ReportAllocs()
	for b.Loop() {
		d.Edit(last, src[last])
	}
}

func BenchmarkEditAllDependents(b *testing.B) {
	src := benchLines()
	d := New()
	checkClean(b, d.SetLines(src))
	if got := len(d.Edit(0, src[0])); got < len(src)-2 {
		b.Fatalf("editing line 0 restaled %d lines, want nearly all %d", got, len(src))
	}
	b.ReportAllocs()
	for b.Loop() {
		d.Edit(0, src[0])
	}
}

func BenchmarkSheetAPI(b *testing.B) {
	req := benchRequest(benchLines())
	var back sheetOut
	if err := json.Unmarshal(EvalRequest(req), &back); err != nil {
		b.Fatalf("EvalRequest: %v", err)
	}
	for _, l := range back.Lines {
		if l.Status == StatusError {
			b.Fatalf("line %d (%s): %s", l.N, l.Text, l.Error)
		}
	}
	b.ReportAllocs()
	for b.Loop() {
		benchSink = EvalRequest(req)
	}
}

func BenchmarkSheetAPIRefusesTooWide(b *testing.B) {
	req := benchRequest(tooLargeToWrite)
	var back sheetOut
	if err := json.Unmarshal(EvalRequest(req), &back); err != nil {
		b.Fatalf("EvalRequest: %v", err)
	}
	for _, l := range back.Lines {
		if l.Status != StatusError {
			b.Fatalf("line %d answered %+v instead of being refused", l.N, l)
		}
	}
	b.ReportAllocs()
	for b.Loop() {
		benchSink = EvalRequest(req)
	}
}
