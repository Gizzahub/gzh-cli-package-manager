package cleanup

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMapFileSystemWalkDirPreservesWrappedSkipDir(t *testing.T) {
	t.Parallel()

	fileSystem := NewMapFileSystem()
	root := "/cache"
	fileSystem.AddFile(filepath.Join(root, "child", "package.tgz"), 1, time.Now())

	want := fmt.Errorf("wrapped skip: %w", fs.SkipDir)
	err := fileSystem.WalkDir(root, func(path string, _ fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == filepath.Join(root, "child") {
			return want
		}
		return nil
	})
	if !errors.Is(err, fs.SkipDir) {
		t.Fatalf("WalkDir() error = %v, want wrapped fs.SkipDir", err)
	}
}

func TestMapFileSystemWalkDirSkipsDescendantsForExactSkipDir(t *testing.T) {
	t.Parallel()

	fileSystem := NewMapFileSystem()
	root := "/cache"
	child := filepath.Join(root, "child")
	descendant := filepath.Join(child, "package.tgz")
	fileSystem.AddFile(descendant, 1, time.Now())

	visitedDescendant := false
	err := fileSystem.WalkDir(root, func(path string, _ fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == child {
			return fs.SkipDir
		}
		if path == descendant {
			visitedDescendant = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir() error = %v, want nil", err)
	}
	if visitedDescendant {
		t.Fatal("WalkDir() visited a descendant after exact fs.SkipDir")
	}
}

func TestOSFileSystemMissingPathPreservesNotExistSemantics(t *testing.T) {
	t.Parallel()

	fileSystem := NewOSFileSystem()
	missingPath := filepath.Join(t.TempDir(), "missing")

	assertNotExist := func(name string, err error) {
		t.Helper()
		if !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("%s() error = %v, want errors.Is(err, fs.ErrNotExist)", name, err)
		}
		if !os.IsNotExist(err) {
			t.Fatalf("%s() error = %v, want os.IsNotExist(err)", name, err)
		}
	}

	_, err := fileSystem.Stat(missingPath)
	assertNotExist("Stat", err)

	err = fileSystem.WalkDir(missingPath, func(_ string, _ fs.DirEntry, walkErr error) error { return walkErr })
	assertNotExist("WalkDir", err)

	_, err = fileSystem.ReadDir(missingPath)
	assertNotExist("ReadDir", err)
}

func TestOSFileSystemRemoveAllMissingPathMatchesOS(t *testing.T) {
	t.Parallel()

	err := NewOSFileSystem().RemoveAll(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatalf("RemoveAll() error = %v, want nil for a missing path", err)
	}
}
