package cleanup

import (
	"errors"
	"fmt"
	"io/fs"
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
