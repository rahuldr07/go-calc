//go:build js && wasm

package main

import (
	"syscall/js"

	"github.com/rahuldr07/go-calc/internal/doc"
)

func main() {
	js.Global().Set("goCalcEval", js.FuncOf(sheet))
	js.Global().Set("goCalcVocabulary", js.FuncOf(vocabulary))
	js.Global().Set("goCalcSteps", js.FuncOf(working))
	select {}
}

func sheet(_ js.Value, args []js.Value) any {
	return string(doc.EvalRequest(requested(args)))
}

func requested(args []js.Value) []byte {
	if len(args) == 0 {
		return nil
	}
	return []byte(args[0].String())
}
