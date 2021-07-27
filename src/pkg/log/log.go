package log

import (
	log "github.com/sirupsen/logrus"
)

func NewMyLog() *log.Logger {
	l := log.New()
	l.SetFormatter(&log.TextFormatter{DisableTimestamp: true})
	return l
}

var Log *log.Logger

func init() {
	Log = NewMyLog()
}
