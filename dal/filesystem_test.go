package dal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileSystemImpl_Exists(t *testing.T) {
	fs := NewFileSystem()

	t.Run("existing file", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "exists-*")
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
		if !fs.Exists(f.Name()) {
			t.Errorf("Exists(%q) = false, want true", f.Name())
		}
	})

	t.Run("non-existing file", func(t *testing.T) {
		if fs.Exists(filepath.Join(t.TempDir(), "no-such-file")) {
			t.Error("Exists returned true for non-existing file")
		}
	})
}

func TestFileSystemImpl_EnsureDir(t *testing.T) {
	fs := NewFileSystem()
	nested := filepath.Join(t.TempDir(), "a", "b", "c")
	if err := fs.EnsureDir(nested); err != nil {
		t.Fatalf("EnsureDir(%q) error: %v", nested, err)
	}
	info, err := os.Stat(nested)
	if err != nil {
		t.Fatalf("Stat(%q) error: %v", nested, err)
	}
	if !info.IsDir() {
		t.Errorf("%q is not a directory", nested)
	}
}

func TestFileSystemImpl_ReadWriteFile(t *testing.T) {
	fs := NewFileSystem()
	path := filepath.Join(t.TempDir(), "roundtrip.txt")
	data := []byte("hello dal")

	if err := fs.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
	got, err := fs.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("ReadFile = %q, want %q", got, data)
	}
}

func TestFileSystemImpl_ReadDir(t *testing.T) {
	fs := NewFileSystem()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)
	os.Mkdir(filepath.Join(dir, "subdir"), 0755)

	entries, err := fs.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("ReadDir returned %d entries, want 2", len(entries))
	}

	names := map[string]bool{}
	for _, e := range entries {
		names[e.Name] = e.IsDir
	}
	if !names["subdir"] {
		t.Error("subdir not found or not marked as dir")
	}
	if names["a.txt"] {
		t.Error("a.txt should not be a directory")
	}
}

func TestFileSystemImpl_BaseName(t *testing.T) {
	fs := NewFileSystem()
	tests := []struct {
		input, want string
	}{
		{"/foo/bar/baz.txt", "baz"},
		{"hello.go", "hello"},
		{"noext", "noext"},
		{"/path/to/.hidden", ""},
	}
	for _, tt := range tests {
		got := fs.BaseName(tt.input)
		if got != tt.want {
			t.Errorf("BaseName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
