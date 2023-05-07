package lib

import (
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
)

func Logger(logFilePath string) zerolog.Logger {
	target := os.Stdout

	// check if a log file config is set
	if logFilePath != "" {
		extension := filepath.Ext(logFilePath)
		path := logFilePath
		if extension == "" {
			path = logFilePath + time.Now().Format("-2006-01-02") + ".log"
		}
		file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		target = file
	}

	return zerolog.New(target).Level(zerolog.InfoLevel).With().Timestamp().Logger()
}
