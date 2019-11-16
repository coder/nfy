// Package lockfile implements a filesystem based mutex.
package lockfile

import (
	"errors"
	"os"
)

var ErrLocked = errors.New("file is locked")

// Lock attempts to lock path. If it is already locked, it returns ErrLocked.
func Lock(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL, 0)
	if err != nil {
		if os.IsExist(err) {
			return ErrLocked
		}
		return err
	}
	f.Close()
	return nil
}

// Unlock unlocks path
func Unlock(path string) {
	_ = os.Remove(path)
}
