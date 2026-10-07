//go:build js && wasm

package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/rahuldr07/go-calc/internal/consts"
	"github.com/rahuldr07/go-calc/internal/eval"
)

type vocabularyOut struct {
	Functions []string          `json:"functions"`
	Constants map[string]string `json:"constants"`
	Units     []string          `json:"units"`
}

func vocabulary(_ js.Value, _ []js.Value) any {
	out := vocabularyOut{
		Functions: eval.Functions(),
		Constants: map[string]string{},
		Units:     eval.Units(),
	}
	for _, symbol := range eval.Constants() {
		if c, ok := consts.Lookup(symbol); ok && c.Name != "" {
			out.Constants[symbol] = c.Name
			continue
		}
		out.Constants[symbol] = "constant"
	}
	text, err := json.Marshal(out)
	if err != nil {
		return ""
	}
	return string(text)
}
