package outputfile

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWritePublishesPrivateFileAndRequiresOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "records.json")
	write := func(w io.Writer) error {
		_, err := io.WriteString(w, "complete\n")
		return err
	}
	if err := Write(path, false, write); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("file mode = %v, want 0600", info.Mode().Perm())
	}
	if err := Write(path, false, func(io.Writer) error {
		t.Fatal("writer ran for an existing output file")
		return nil
	}); err == nil || !strings.Contains(err.Error(), "--overwrite") {
		t.Fatalf("existing file error = %v", err)
	}
	if err := Write(path, true, func(w io.Writer) error {
		_, err := io.WriteString(w, "replacement\n")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	assertContent(t, path, "replacement\n")
	assertNoTemps(t, dir)
}

func TestWriteFailurePreservesDestination(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "new", true: "existing"}[existing], func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "records.csv")
			if existing {
				if err := os.WriteFile(path, []byte("original\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			wantErr := errors.New("encoding failed")
			err := Write(path, true, func(w io.Writer) error {
				if _, err := io.WriteString(w, "partial\n"); err != nil {
					return err
				}
				return wantErr
			})
			if !errors.Is(err, wantErr) {
				t.Fatalf("Write() = %v, want %v", err, wantErr)
			}
			if existing {
				assertContent(t, path, "original\n")
			} else if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("failed output exists: %v", err)
			}
			assertNoTemps(t, dir)
		})
	}
}

func TestWriteCloseFailurePreservesDestination(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "records.jsonl")
	if err := os.WriteFile(path, []byte("original\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := Write(path, true, func(w io.Writer) error {
		return w.(io.Closer).Close()
	})
	if err == nil || !strings.Contains(err.Error(), "close output file") {
		t.Fatalf("close error = %v", err)
	}
	assertContent(t, path, "original\n")
	assertNoTemps(t, dir)
}

func TestWriteDoesNotReplaceFileCreatedDuringEncoding(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "records.json")
	err := Write(path, false, func(w io.Writer) error {
		if _, err := io.WriteString(w, "new\n"); err != nil {
			return err
		}
		return os.WriteFile(path, []byte("competing writer\n"), 0o600)
	})
	if err == nil || !strings.Contains(err.Error(), "--overwrite") {
		t.Fatalf("collision error = %v", err)
	}
	assertContent(t, path, "competing writer\n")
	assertNoTemps(t, dir)
}

func TestWriteDoesNotFollowDestinationSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.json")
	path := filepath.Join(dir, "records.json")
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	write := func(w io.Writer) error {
		_, err := io.WriteString(w, "new\n")
		return err
	}
	if err := Write(path, false, write); err == nil {
		t.Fatal("Write() replaced a dangling symlink")
	}
	if err := Write(path, true, write); err != nil {
		t.Fatal(err)
	}
	assertContent(t, path, "new\n")
	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Write() followed the symlink: %v", err)
	}
	assertNoTemps(t, dir)
}

func assertContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("%s = %q, want %q", path, got, want)
	}
}

func assertNoTemps(t *testing.T, dir string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, ".ynab-output-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files remain: %v", matches)
	}
}
