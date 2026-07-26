package main

import "testing"

func TestHelpDoesNotRequireInstalledCredentials(t *testing.T) {
	t.Setenv("BRIDGE_ADMIN_TOKEN", "")
	t.Setenv("BRIDGE_OPERATOR_ENV", t.TempDir()+"/missing.env")
	if err := run([]string{"--help"}); err != nil {
		t.Fatalf("help required broker credentials: %v", err)
	}
}
