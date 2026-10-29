// Package outputfile publishes complete output without exposing partial files.
package outputfile

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Write writes into a private temporary file beside path and publishes it only
// after write and Close succeed. Unless overwrite is true, an existing path,
// including a dangling symlink or one created during write, is never replaced.
func Write(path string, overwrite bool, write func(io.Writer) error) error {
	if !overwrite {
		if _, err := os.Lstat(path); err == nil {
			return fmt.Errorf("output file %q exists; use --overwrite to replace it", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("check output file: %w", err)
		}
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".ynab-output-*")
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	temp := file.Name()
	defer func() { _ = os.Remove(temp) }()
	err = write(file)
	closeErr := file.Close()
	if err != nil {
		return fmt.Errorf("write output file: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("close output file: %w", closeErr)
	}
	if overwrite {
		err = os.Rename(temp, path)
	} else {
		// A hard link publishes the same completed file atomically and fails
		// if the destination appeared since the initial check.
		err = os.Link(temp, path)
	}
	if errors.Is(err, os.ErrExist) {
		return fmt.Errorf("output file %q exists; use --overwrite to replace it", path)
	}
	if err != nil {
		return fmt.Errorf("publish output file: %w", err)
	}
	return nil
}
