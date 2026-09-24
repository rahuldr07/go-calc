package eval

import (
	"strings"
	"testing"

	"github.com/rahuldr07/go-calc/internal/parse"
)

func TestAddingTwoAbsoluteTemperaturesIsRefused(t *testing.T) {
	env := NewEnv()
	n, _, err := parse.Parse("20 degC + 30 degC")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = env.Eval(n)
	if err == nil {
		t.Fatal("want error adding two absolute temperatures")
	}
	if !strings.Contains(err.Error(), "absolute temperatures") {
		t.Errorf("error = %q, want it to explain absolute temperatures", err)
	}
}

func TestTemperaturePlusDifference(t *testing.T) {
	env := NewEnv()
	if got := run(t, env, "20 degC + 5 K"); got != "25 degC" {
		t.Errorf("20 degC + 5 K = %s, want 25 degC", got)
	}
}
