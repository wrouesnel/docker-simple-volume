package volumesetup

import (
	"os"
	"io"
)

// WriteAndSyncExistingFile writes a file and calls sync, ensuring data is
// written to the device if it exits successfully.
func WriteAndSyncExistingFile(filename string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	n, err := f.Write(data)
	if err == nil && n < len(data) {
		err = io.ErrShortWrite
	}
	if err == nil {
		err = f.Sync()
	}
	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}