// +build !windows

package logger

import "path/filepath"

func registerOSSinks() error {
	return nil
}

func logFileURI(configRootDir string) string {
	return filepath.ToSlash(filepath.Join(configRootDir, "log.jsonl"))
}
