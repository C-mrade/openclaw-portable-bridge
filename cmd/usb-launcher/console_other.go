//go:build !linux

package main

func ensureInteractiveConsole(string) (bool, error) {
	return false, nil
}
