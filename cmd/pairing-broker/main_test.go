package main

import "testing"

func TestDeveloperCapabilityProfileIsAccepted(t *testing.T) {
	developer := []string{
		"system.info", "system.network", "disk.list", "service.list",
		"process.list", "process.start", "process.stop-owned", "shell.run",
		"shell.run-admin", "powershell.run", "shell.start", "shell.status",
		"shell.cancel", "files.list", "files.read", "files.read-chunk",
		"files.write", "files.write-chunk", "files.upload", "files.download",
		"session.disconnect",
	}
	if !validCapabilities(developer) {
		t.Fatalf("Developer profile with %d capabilities was rejected", len(developer))
	}
}

func TestCapabilityValidationRejectsUnknownDuplicateAndOversized(t *testing.T) {
	if validCapabilities([]string{"system.info", "unknown"}) {
		t.Fatal("unknown capability accepted")
	}
	if validCapabilities([]string{"system.info", "system.info"}) {
		t.Fatal("duplicate capability accepted")
	}
	tooMany := make([]string, 33)
	for i := range tooMany {
		tooMany[i] = "system.info"
	}
	if validCapabilities(tooMany) {
		t.Fatal("oversized capability request accepted")
	}
}
