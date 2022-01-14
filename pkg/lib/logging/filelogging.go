package logging

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

func GetLoggingFile(path string) (*os.File, error) {
	extension := filepath.Ext(path)
	if extension != "" {
		path = strings.Replace(path, extension, time.Now().Format("2006-01-02 15:04:05")+extension, 1)
	} else {
		path = path + time.Now().Format("2006-01-02 15:04:05")
	}

	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	return f, err
}
