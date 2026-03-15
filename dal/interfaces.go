package dal

import "io"

// FileSystem abstracts file and directory operations.
type FileSystem interface {
	Exists(path string) bool
	EnsureDir(dir string) error
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm int) error
	ReadDir(path string) ([]DirEntry, error)
	BaseName(path string) string
}

// DirEntry is a minimal directory entry.
type DirEntry struct {
	Name  string
	IsDir bool
}

// Executor abstracts command execution and PATH lookups.
type Executor interface {
	Run(name string, args ...string) (string, error)
	RunInDir(dir, name string, args ...string) (string, error)
	RunOsascript(script string) string
	Which(cmd string) bool
}

// Logger abstracts structured logger initialization.
type Logger interface {
	Init(verbose bool, w io.Writer)
}
