package genrenorm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUndoSessionRestoresLatestBackup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	file := filepath.Join(t.TempDir(), "track.mp3")
	if err := os.WriteFile(file, []byte("before"), 0644); err != nil {
		t.Fatal(err)
	}

	session, err := StartUndoSession("test-tool")
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Backup(file); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("after"), 0644); err != nil {
		t.Fatal(err)
	}

	count, err := RestoreLatestUndo("test-tool")
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("restore count = %d, want 1", count)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "before" {
		t.Fatalf("restored content = %q, want before", string(data))
	}
}
