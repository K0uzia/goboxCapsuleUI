//go:build !linux

package gui

func grabKeyboard(func(code int)) (func(), bool) { return func() {}, false }

func keySymbol(int) string { return "" }
