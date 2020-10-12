package log

import (
	"fmt"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"os"
)

var logger *logrus.Logger

func init() {
	var cfg = struct {
		LogLevel string `default:"INFO" split_words:"true"`
	}{}
	os.Environ()
	err := envconfig.Process("traebeler", &cfg)
	if err != nil {
		panic("failed processing logger environment variables. Error: " + err.Error())
	}

	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		panic(fmt.Sprintf("failed parsing log level '%s' as a valid log level", cfg.LogLevel))
	}

	logger = &logrus.Logger{
		Out: os.Stderr,
		Formatter: new(logrus.JSONFormatter),
		Hooks: make(logrus.LevelHooks),
		Level: level,
	}
}

// Embodies all relevant functions from https://github.com/sirupsen/logrus/blob/master/exported.go

// Debugf logs a message at level Debug on the standard logger.
func Debugf(format string, args ...interface{}) {
	logger.Debugf(format, args...)
}

// Printf logs a message at level Info on the standard logger.
func Printf(format string, args ...interface{}) {
	logger.Printf(format, args...)
}

// Info logs a message at level Info on the standard logger.
func Info(args ...interface{}) {
	logger.Info(args...)
}

// Panic logs a message at level Panic on the standard logger.
func Panic(args ...interface{}) {
	logger.Panic(args...)
}

// Infof logs a message at level Info on the standard logger.
func Infof(format string, args ...interface{}) {
	logger.Infof(format, args...)
}

// Errorf logs a message at level Error on the standard logger.
func Errorf(format string, args ...interface{}) {
	logger.Errorf(format, args...)
}

// Panicf logs a message at level Panic on the standard logger.
func Panicf(format string, args ...interface{}) {
	logger.Panicf(format, args...)
}