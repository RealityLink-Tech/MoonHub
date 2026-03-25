package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureLANDeviceIDStable(t *testing.T) {
	dir := t.TempDir()
	id1, err := EnsureLANDeviceID(dir)
	if err != nil {
		t.Fatal(err)
	}
	if id1 == "" {
		t.Fatal("empty id")
	}
	id2, err := EnsureLANDeviceID(dir)
	if err != nil {
		t.Fatal(err)
	}
	if id1 != id2 {
		t.Fatalf("expected same id, got %q and %q", id1, id2)
	}
	p := filepath.Join(dir, lanDeviceIDFileName)
	if _, err := os.Stat(p); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureLANDeviceIDEmptyDir(t *testing.T) {
	_, err := EnsureLANDeviceID("")
	if err == nil {
		t.Fatal("expected error for empty config dir")
	}
}
