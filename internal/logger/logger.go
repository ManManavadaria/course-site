package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

// Init initializes the logger with proper configuration
func Init() {
	// Set log format to JSON for better parsing
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// Set output to stdout
	logrus.SetOutput(os.Stdout)

	// Set log level based on environment
	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		level = "info" // default level
	}

	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logrus.SetLevel(logLevel)

	// Add caller information
	logrus.SetReportCaller(true)

	logrus.Info("Logger initialized")
}
