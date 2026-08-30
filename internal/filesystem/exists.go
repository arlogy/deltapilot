package filesystem

import (
	"os"
)

// ExistsDir reports whether path exists and is a directory.
func ExistsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// ExistsFile reports whether path exists and is a regular file.
func ExistsFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
