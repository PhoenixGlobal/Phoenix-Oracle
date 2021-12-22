// +build windows

package logger

import (
	"net/url"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

func logFileURI(configRootDir string) string {
	return "winfile:///" + filepath.ToSlash(filepath.Join(configRootDir, "log.jsonl"))
}

func registerOSSinks() error {
	return zap.RegisterSink("winfile", newWinFileSink)
}

func newWinFileSink(u *url.URL) (zap.Sink, error) {
	return os.OpenFile(u.Path[1:], os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
}
