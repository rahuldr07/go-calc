package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func calcFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sheet.calc")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCheckPassesACleanSheet(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "")
	path := calcFile(t, "# trip\ndistance = 412 km\nr = 2\narea = pi * r^2\n")
	out, code := check([]string{path}, settings{})
	if code != 0 || !strings.Contains(out, "3 lines (2 exact, 1 proven) — ok") {
		t.Fatalf("exit %d: %s", code, out)
	}
}

func TestCheckFailsARefusalWithAnAnnotation(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "")
	path := calcFile(t, "x = 3\ny = x + z\n")
	out, code := check([]string{"--github", path}, settings{})
	if code != 1 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if !strings.Contains(out, "::error file="+path+",line=2,col=9::refused: unknown name z") || !strings.Contains(out, "1 refused") {
		t.Errorf("output: %s", out)
	}
}

func TestCheckHoldsLinesToTheRequiredTrust(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "")
	path := calcFile(t, "1/3\nsqrt(2)\nf(x) = x^2\n")
	if out, code := check([]string{"--require=proven", path}, settings{}); code != 0 {
		t.Errorf("proven: exit %d: %s", code, out)
	}
	out, code := check([]string{"--require=exact", path}, settings{})
	if code != 1 || !strings.Contains(out, ":2: sqrt(2) is proven, but exact is required") || strings.Contains(out, ":3:") {
		t.Errorf("exact: exit %d: %s", code, out)
	}
}

func TestCheckComparesAgainstTheGoldenFile(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "")
	path := calcFile(t, "a = 412 km\na in mile\n")
	if out, code := check([]string{"--update", path}, settings{}); code != 0 || !strings.Contains(out, "golden file written") {
		t.Fatalf("update: exit %d: %s", code, out)
	}
	golden, err := os.ReadFile(path + goldenSuffix)
	if err != nil || !strings.HasPrefix(string(golden), "1\texact\t412 km\n") {
		t.Fatalf("golden = %q, %v", golden, err)
	}
	if out, code := check([]string{path}, settings{}); code != 0 {
		t.Fatalf("unchanged: exit %d: %s", code, out)
	}
	if err := os.WriteFile(path, []byte("a = 400 km\na in mile\n2 + 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := check([]string{path}, settings{})
	if code != 1 || !strings.Contains(out, ":1: changed from 412 km (exact) to 400 km (exact)") || !strings.Contains(out, ":3: new answer 4 (exact)") {
		t.Errorf("drift: exit %d: %s", code, out)
	}
}

func TestCheckRejectsBadUsage(t *testing.T) {
	if _, code := check(nil, settings{}); code != 2 {
		t.Errorf("no files: exit %d", code)
	}
	if out, code := check([]string{"--require=sure", "x.calc"}, settings{}); code != 2 || !strings.Contains(out, "--require takes") {
		t.Errorf("bad level: exit %d: %s", code, out)
	}
	if out, code := check([]string{"/no/such/file.calc"}, settings{}); code != 1 || !strings.Contains(out, "no such file") {
		t.Errorf("missing file: exit %d: %s", code, out)
	}
}
