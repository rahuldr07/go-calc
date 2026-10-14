package eval

import (
	"slices"
	"testing"

	"github.com/rahuldr07/go-calc/internal/parse"
)

func TestEveryFunctionIsDocumentedWithAWorkingExample(t *testing.T) {
	names := Functions()
	for _, name := range names {
		doc, ok := Docs()[name]
		if !ok {
			t.Errorf("%s has no entry in docs.go, so help cannot describe it", name)
			continue
		}
		if doc.Signature == "" || doc.Summary == "" || doc.Example == "" {
			t.Errorf("%s has an incomplete entry: %+v", name, doc)
		}
		if !slices.Contains(CategoryOrder, doc.Category) {
			t.Errorf("%s is in an unknown category %q", name, doc.Category)
		}
		node, _, err := parse.Parse(doc.Example)
		if err != nil {
			t.Errorf("%s example %q does not parse: %v", name, doc.Example, err)
			continue
		}
		if _, err := NewEnv().Eval(node); err != nil {
			t.Errorf("%s example %q fails: %v", name, doc.Example, err)
		}
	}
	for name := range Docs() {
		if !slices.Contains(names, name) {
			t.Errorf("docs.go describes %s, which is not a function", name)
		}
	}
}
