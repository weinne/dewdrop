package config

import (
	"path/filepath"
	"testing"
)

func TestRemotePathStore_GetFallback(t *testing.T) {
	tmp := t.TempDir()
	store := NewRemotePathStore(filepath.Join(tmp, "remote-paths.conf"), "/home/test/Nuvem")

	got, err := store.Get("drive")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	want := "/home/test/Nuvem/drive"
	if got != want {
		t.Fatalf("Get fallback mismatch: got %q want %q", got, want)
	}
}

func TestRemotePathStore_SetAndGet(t *testing.T) {
	tmp := t.TempDir()
	store := NewRemotePathStore(filepath.Join(tmp, "remote-paths.conf"), "/home/test/Nuvem")

	if err := store.Set("drive", "/mnt/data/drive"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	got, err := store.Get("drive")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	if got != "/mnt/data/drive" {
		t.Fatalf("Get value mismatch: got %q", got)
	}
}

func TestRemotePathStore_PathWithPipe(t *testing.T) {
	tmp := t.TempDir()
	store := NewRemotePathStore(filepath.Join(tmp, "remote-paths.conf"), "/home/test/Nuvem")

	pathWithPipe := "/mnt/data/one|two"
	if err := store.Set("onedrive", pathWithPipe); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	got, err := store.Get("onedrive")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	if got != pathWithPipe {
		t.Fatalf("Get value mismatch: got %q want %q", got, pathWithPipe)
	}
}
