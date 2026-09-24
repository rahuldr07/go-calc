package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildGocalc(b *testing.B) string {
	b.Helper()
	bin := filepath.Join(b.TempDir(), "gocalc")
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	if err != nil {
		b.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

func BenchmarkColdStart(b *testing.B) {
	bin := buildGocalc(b)
	out, err := exec.Command(bin, "0.1 + 0.2").Output()
	if err != nil {
		b.Fatalf("gocalc: %v", err)
	}
	if strings.TrimSpace(string(out)) != "0.3" {
		b.Fatalf("gocalc answered %q", out)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := exec.Command(bin, "0.1 + 0.2").Run(); err != nil {
			b.Fatalf("gocalc: %v", err)
		}
	}
}

func BenchmarkColdStartTypicalExpression(b *testing.B) {
	bin := buildGocalc(b)
	if err := exec.Command(bin, "2 * sin(pi/4) + sqrt(2) / 3").Run(); err != nil {
		b.Fatalf("gocalc: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := exec.Command(bin, "2 * sin(pi/4) + sqrt(2) / 3").Run(); err != nil {
			b.Fatalf("gocalc: %v", err)
		}
	}
}
