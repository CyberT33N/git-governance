//go:build windows

package configfs

import (
	"errors"
	"os"
)

func recoverConfiguration(path string) error {
	backup := path + ".bak"
	_, targetErr := os.Stat(path)
	switch {
	case targetErr == nil:
		if err := os.Remove(backup); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	case !errors.Is(targetErr, os.ErrNotExist):
		return targetErr
	}

	if _, err := os.Stat(backup); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return os.Rename(backup, path)
}

func replaceConfiguration(path, temporaryPath string) (returnErr error) {
	if err := recoverConfiguration(path); err != nil {
		return err
	}

	backup := path + ".bak"
	hadTarget := false
	if _, err := os.Stat(path); err == nil {
		hadTarget = true
		if err := os.Rename(path, backup); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if err := os.Rename(temporaryPath, path); err != nil {
		if hadTarget {
			_ = os.Rename(backup, path)
		}
		return err
	}
	if hadTarget {
		_ = os.Remove(backup)
	}
	return nil
}
