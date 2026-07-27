//go:build linux

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

const consoleChildEnvironment = "OPENCLAW_BRIDGE_CONSOLE_CHILD=1"

type terminalCandidate struct {
	name string
	args func(string) []string
}

var linuxTerminalCandidates = []terminalCandidate{
	{name: "xdg-terminal-exec", args: func(self string) []string { return []string{self} }},
	{name: "kitty", args: func(self string) []string { return []string{"--title", "OpenClaw Portable Bridge", self} }},
	{name: "foot", args: func(self string) []string { return []string{"--title=OpenClaw Portable Bridge", self} }},
	{name: "alacritty", args: func(self string) []string { return []string{"--title", "OpenClaw Portable Bridge", "-e", self} }},
	{name: "gnome-terminal", args: func(self string) []string { return []string{"--wait", "--", self} }},
	{name: "kgx", args: func(self string) []string { return []string{"--", self} }},
	{name: "konsole", args: func(self string) []string { return []string{"-e", self} }},
	{name: "x-terminal-emulator", args: func(self string) []string { return []string{"-T", "OpenClaw Portable Bridge", "-e", self} }},
	{name: "xterm", args: func(self string) []string { return []string{"-T", "OpenClaw Portable Bridge", "-e", self} }},
}

func ensureInteractiveConsole(self string) (bool, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err == nil {
		_ = tty.Close()
		return false, nil
	}
	if os.Getenv("OPENCLAW_BRIDGE_CONSOLE_CHILD") == "1" {
		return false, errors.New("the selected terminal emulator did not provide an interactive console")
	}
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return false, errors.New("an interactive terminal is required; run this launcher from a visible terminal")
	}

	path, args, err := linuxTerminalCommand(self, exec.LookPath)
	if err != nil {
		return false, err
	}
	cmd := exec.Command(path, args...)
	cmd.Env = append(os.Environ(), consoleChildEnvironment)
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("open visible terminal with %s: %w", path, err)
	}
	return true, nil
}

func linuxTerminalCommand(self string, lookPath func(string) (string, error)) (string, []string, error) {
	for _, candidate := range linuxTerminalCandidates {
		path, err := lookPath(candidate.name)
		if err == nil {
			return path, candidate.args(self), nil
		}
	}
	return "", nil, errors.New("no supported terminal emulator found; install xdg-terminal-exec, Kitty, Foot, Alacritty, GNOME Terminal, Console, Konsole, or xterm")
}
