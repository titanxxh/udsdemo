package mlog

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// global logger
var L = logrus.NewEntry(&logrus.Logger{
	Out: os.Stderr,
	Formatter: &logrus.TextFormatter{
		TimestampFormat: time.RFC3339Nano,
	},
	Hooks: make(logrus.LevelHooks),
	Level: logrus.DebugLevel,
})
