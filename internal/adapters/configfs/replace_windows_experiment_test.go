//go:build windows

package configfs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

const fsctlRequestOplockLevel1 = 0x00090000

func TestExperimentWindowsReplacementFailureModes(t *testing.T) {
	directory := t.TempDir()

	t.Run("invalid path", func(t *testing.T) {
		path := filepath.Join(directory, "invalid\x00name")
		_, err := os.Stat(path)
		t.Logf("Stat(%q) error = %#v", path, err)
	})

	t.Run("locked symlink", func(t *testing.T) {
		target := filepath.Join(directory, "symlink-target")
		link := filepath.Join(directory, "symlink")
		if err := os.WriteFile(target, []byte("target"), defaultFileMode); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Base(target), link); err != nil {
			t.Fatal(err)
		}
		handle, err := openReparsePointWithoutDeleteShare(link)
		if err != nil {
			t.Fatal(err)
		}
		defer windows.CloseHandle(handle)

		_, err = os.Stat(link)
		t.Logf("Stat(%q) while locked error = %#v", link, err)
	})

	t.Run("deleted symlink target", func(t *testing.T) {
		path := filepath.Join(directory, "config.json")
		backup := path + ".bak"
		if err := os.WriteFile(backup, []byte("backup"), defaultFileMode); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Base(backup), path); err != nil {
			t.Skipf("create symbolic link: %v", err)
		}
		backupPointer, err := windows.UTF16PtrFromString(backup)
		if err != nil {
			t.Fatal(err)
		}
		handle, err := windows.CreateFile(
			backupPointer,
			windows.GENERIC_READ,
			windows.FILE_SHARE_DELETE,
			nil,
			windows.OPEN_EXISTING,
			windows.FILE_ATTRIBUTE_NORMAL,
			0,
		)
		if err != nil {
			t.Fatal(err)
		}
		defer windows.CloseHandle(handle)

		_, err = os.Stat(path)
		t.Logf("Stat(%q) before recovery error = %#v", path, err)
		err = recoverConfiguration(path)
		t.Logf("recoverConfiguration(%q) error = %#v", path, err)
		_, err = os.Stat(path)
		t.Logf("Stat(%q) after recovery error = %#v", path, err)
	})

	t.Run("deleted junction target", func(t *testing.T) {
		path := filepath.Join(directory, "junction-config")
		backup := path + ".bak"
		if err := os.Mkdir(backup, defaultDirectoryMode); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Base(backup), path); err != nil {
			t.Fatal(err)
		}
		err := replaceConfiguration(path, filepath.Join(directory, "unused-junction.tmp"))
		t.Logf("replaceConfiguration(%q) error = %#v, not exists = %t", path, err, errors.Is(err, os.ErrNotExist))
	})

	t.Run("oplock coordinates lock after recovery", func(t *testing.T) {
		path := filepath.Join(directory, "oplock-config.json")
		backup := path + ".bak"
		if err := os.WriteFile(path, []byte("target"), defaultFileMode); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(backup, []byte("stale"), defaultFileMode); err != nil {
			t.Fatal(err)
		}

		backupPointer, err := windows.UTF16PtrFromString(backup)
		if err != nil {
			t.Fatal(err)
		}
		backupHandle, err := windows.CreateFile(
			backupPointer,
			windows.GENERIC_READ,
			windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
			nil,
			windows.OPEN_EXISTING,
			windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_OVERLAPPED,
			0,
		)
		if err != nil {
			t.Fatal(err)
		}
		event, err := windows.CreateEvent(nil, 0, 0, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer windows.CloseHandle(event)
		overlapped := windows.Overlapped{HEvent: event}
		err = windows.DeviceIoControl(backupHandle, fsctlRequestOplockLevel1, nil, 0, nil, 0, nil, &overlapped)
		if !errors.Is(err, windows.ERROR_IO_PENDING) {
			t.Fatalf("request oplock = %v, want ERROR_IO_PENDING", err)
		}

		type result struct {
			err error
		}
		locked := make(chan result, 1)
		go func() {
			defer windows.CloseHandle(backupHandle)
			status, waitErr := windows.WaitForSingleObject(event, 5_000)
			if waitErr != nil {
				locked <- result{err: waitErr}
				return
			}
			if status != windows.WAIT_OBJECT_0 {
				locked <- result{err: fmt.Errorf("oplock wait status = %#x", status)}
				return
			}
			if err := os.Remove(path); err != nil {
				locked <- result{err: err}
				return
			}
			if err := os.Symlink(filepath.Base(path), path); err != nil {
				locked <- result{err: err}
				return
			}
			locked <- result{}
		}()

		err = replaceConfiguration(path, filepath.Join(directory, "unused.tmp"))
		lockedResult := <-locked
		if cleanupErr := os.Remove(path); cleanupErr != nil {
			t.Fatal(cleanupErr)
		}
		t.Logf("replaceConfiguration error = %#v, not exists = %t, oplock result = %#v", err, errors.Is(err, os.ErrNotExist), lockedResult.err)
	})
}

func openReparsePointWithoutDeleteShare(path string) (windows.Handle, error) {
	pointer, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	handle, err := windows.CreateFile(
		pointer,
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_OPEN_REPARSE_POINT,
		0,
	)
	if err != nil {
		return 0, err
	}
	return handle, nil
}
