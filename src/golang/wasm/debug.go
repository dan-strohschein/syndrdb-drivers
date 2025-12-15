//go:build js && wasm
// +build js,wasm

package main

import (
"fmt"
"syscall/js"
)

func debugConnect(this js.Value, args []js.Value) interface{} {
	fmt.Printf("DEBUG connect called: len(args)=%d\n", len(args))
	for i, arg := range args {
		fmt.Printf("  args[%d]: Type=%s, String=%s\n", i, arg.Type(), arg.String())
	}
	return js.ValueOf("debug")
}
