package dal

import (
	"os"
	"path/filepath"
	"strings"
)

// FileSystemImpl is the real OS-backed FileSystem.
type FileSystemImpl struct{}

// NewFileSystem returns a new FileSystemImpl.
func NewFileSystem() *FileSystemImpl { return &FileSystemImpl{} }

func (f *FileSystemImpl) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (f *FileSystemImpl) EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

func (f *FileSystemImpl) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (f *FileSystemImpl) WriteFile(path string, data []byte, perm int) error {
	return os.WriteFile(path, data, os.FileMode(perm))
}

func (f *FileSystemImpl) ReadDir(path string) ([]DirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	result := make([]DirEntry, len(entries))
	for i, e := range entries {
		result[i] = DirEntry{Name: e.Name(), IsDir: e.IsDir()}
	}
	return result, nil
}

func (f *FileSystemImpl) BaseName(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}
