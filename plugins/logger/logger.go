package logger

import (
	"os"

	joonix "github.com/joonix/log"
	"github.com/shoplineapp/go-app/plugins"
	"github.com/shoplineapp/go-app/plugins/env"
	"github.com/sirupsen/logrus"
)

func init() {
	plugins.Registry = append(plugins.Registry, NewLogger)
}

var logger *Logger

type Logger struct {
	logrus.Logger
}

type Fields map[string]interface{}

func NewLogger(env *env.Env) *Logger {
	l := logrus.Logger{
		Out:          os.Stderr,
		Formatter:    new(logrus.TextFormatter),
		Hooks:        make(logrus.LevelHooks),
		Level:        logrus.DebugLevel,
		ExitFunc:     os.Exit,
		ReportCaller: false,
	}

	if env.GetEnv("ENVIRONMENT") == "production" || env.GetEnv("LOG_TO_CLOUDWATCH") == "true" {
		l.SetFormatter(joonix.NewFormatter())
		l.SetReportCaller(true)
	}

	switch level := env.GetEnv("LOG_LEVEL"); level {
	case "trace":
		l.SetLevel(logrus.TraceLevel)
	case "debug":
		l.SetLevel(logrus.DebugLevel)
	case "info":
		l.SetLevel(logrus.InfoLevel)
	default:
		if env.GetEnv("ENVIRONMENT") == "production" {
			l.SetLevel(logrus.InfoLevel)
		} else {
			l.SetLevel(logrus.DebugLevel)
		}
	}
	l.SetOutput(os.Stdout)

	logger = &Logger{
		Logger: l,
	}

	return logger
}
