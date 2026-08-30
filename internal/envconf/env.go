package envconf

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/arlogy/deltapilot/internal/workspace"
	"github.com/joho/godotenv"
)

func LoadDotEnv() error {
	rootDir, err := workspace.ProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to load env file: %v", err)
	}

	// https://github.com/joho/godotenv#usage
	err = godotenv.Load(filepath.Join(rootDir, ".env"))
	if err != nil {
		return fmt.Errorf("failed to load env file: %v", err)
	}

	return nil
}

func GetEnvVar(name string) string {
	return os.Getenv(name) // after godotenv.Load()
}
