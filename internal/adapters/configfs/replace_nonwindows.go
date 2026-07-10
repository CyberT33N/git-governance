//go:build !windows

package configfs

import "os"

func recoverConfiguration(string) error {
	return nil
}

func replaceConfiguration(path, temporaryPath string) error {
	return os.Rename(temporaryPath, path)
}
