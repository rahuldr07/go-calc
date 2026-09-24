package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahuldr07/go-calc/internal/format"
)

var plain = format.Style{Digits: format.Digits}

func TestRunOneShot(t *testing.T) {
	out, code := run([]string{"500 + 10%"}, plain, false)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if strings.TrimSpace(out) != "550" {
		t.Errorf("out = %q, want 550", out)
	}
}

func TestRunShowsInterpretation(t *testing.T) {
	out, code := run([]string{"1/2m"}, plain, false)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if !strings.Contains(out, "read as (1/2)·m") {
		t.Errorf("out = %q, want the interpretation note", out)
	}
}

func TestRunJSON(t *testing.T) {
	out, code := run([]string{"3", "km", "to", "m"}, plain, true)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if !strings.Contains(out, `"schema": "go-calc.result/1"`) {
		t.Errorf("out = %q", out)
	}
}

func TestRunErrorExitsNonZero(t *testing.T) {
	out, code := run([]string{"5 m + 3 s"}, plain, false)
	if code == 0 {
		t.Fatal("want non-zero exit")
	}
	if !strings.Contains(out, "cannot add length (m) and time (s)") {
		t.Errorf("out = %q", out)
	}
}

func TestRunFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sheet.calc")
	body := "# lab\nh = 3 m\nh + 200 cm\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := runFile(path, plain, false)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if !strings.Contains(out, "= 5 m") {
		t.Errorf("out = %q, want 5 m", out)
	}
}

func TestUsageWithoutArguments(t *testing.T) {
	out, code := run(nil, plain, false)
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
	if !strings.Contains(out, "usage:") {
		t.Errorf("out = %q", out)
	}
}

func TestRunFileRejectsAForwardReference(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sheet.calc")
	body := "total = price * 2\nprice = 5\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := runFile(path, plain, false)
	if code == 0 {
		t.Fatalf("want non-zero exit, got %q", out)
	}
	if !strings.Contains(out, "line 1: uses price before it is defined") {
		t.Errorf("out = %q, want the forward-reference error on line 1", out)
	}
	if strings.Contains(out, "= 10") {
		t.Errorf("out = %q, want no answer for the forward reference", out)
	}
}

func TestRunFileReportsACycle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sheet.calc")
	body := "a = b + 1\nb = a + 1\nz = 7\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := runFile(path, plain, false)
	if code == 0 {
		t.Fatalf("want non-zero exit, got %q", out)
	}
	if !strings.Contains(out, "line 1: this line depends on itself") {
		t.Errorf("out = %q, want the cycle reported on line 1", out)
	}
	if !strings.Contains(out, "= 7") {
		t.Errorf("out = %q, want the independent line still evaluated", out)
	}
}

func TestRunFileExplainsADependentOfAFailedLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sheet.calc")
	body := "bad = 5 m + 3 s\nuses = bad + 1\nfine = 1 + 1\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := runFile(path, plain, false)
	if code == 0 {
		t.Fatalf("want non-zero exit, got %q", out)
	}
	if !strings.Contains(out, "line 2: uses bad, which failed on line 1") {
		t.Errorf("out = %q, want the dependent to name the line it depends on", out)
	}
	if strings.Contains(out, "unknown name bad") {
		t.Errorf("out = %q, leaked a raw evaluator error", out)
	}
	if !strings.Contains(out, "= 2") {
		t.Errorf("out = %q, want the independent line still evaluated", out)
	}
}
