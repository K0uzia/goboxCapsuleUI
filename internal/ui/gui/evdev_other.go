//go:build !linux

package gui

func readEvdevKeys(string, func(code int)) (func(), []string) { return func() {}, nil }
