package logging

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/gommon/log"
	"github.com/ziflex/lecho/v3"
)

func Logger(logFilePath string) *lecho.Logger {
	logger := lecho.New(
		os.Stdout, // default to STDOUT
		lecho.WithLevel(log.DEBUG),
		lecho.WithTimestamp(),
	)
	// check if a log file config is set
	if logFilePath != "" {
		file, err := GetLoggingFile(logFilePath)
		if err != nil {
			logger.Error("failed to create logging file: %v", err)
		}
		logger.SetOutput(file)
	}

	return logger
}

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
