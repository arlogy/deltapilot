package workspace

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/arlogy/deltapilot/internal/filesystem"
)

// ProjectRoot computes the root directory of the project containing this package.
func ProjectRoot() (string, error) {
	_, lineFilePath, _, ok := runtime.Caller(0) // get the source location of this line
	if !ok {
		return "", errors.New("cannot determine source location")
	}

	projectDirPath := lineFilePath
	for i := 1; i <= 3; i++ {
		projectDirPath = filepath.Dir(projectDirPath)
	}
	if !isProjectDir(projectDirPath) {
		return "", fmt.Errorf("computed project directory does not meet requirements: %s", projectDirPath)
	}

	return projectDirPath, nil
}

func isProjectDir(dirPath string) bool {
	if !filesystem.ExistsDir(dirPath) {
		return false
	}

	requiredDirs := []string{".git", "internal/workspace"}
	for _, dirName := range requiredDirs {
		requiredPath := filepath.Join(dirPath, dirName)
		if !filesystem.ExistsDir(requiredPath) {
			return false
		}
	}

	requiredFiles := []string{"migrations/mariadb_mysql/001_create_chronicle_snapshot_table.sql"}
	for _, dirName := range requiredFiles {
		requiredPath := filepath.Join(dirPath, dirName)
		if !filesystem.ExistsFile(requiredPath) {
			return false
		}
	}

	return true
}
