//go:build js && wasm

package main

import (
	"syscall/js"

	"github.com/rahuldr07/go-calc/internal/steps"
)

func working(_ js.Value, args []js.Value) any {
	if len(args) < 2 {
		return ""
	}
	out, err := steps.FromLine(args[0].String(), []byte(args[1].String()))
	if err != nil {
		return ""
	}
	return string(out)
}
