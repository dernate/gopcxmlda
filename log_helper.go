package gopcxmlda

import (
	log "github.com/sirupsen/logrus"
)

func LogLevel(level uint32) {
	log.SetLevel(log.Level(level))
}

func logDebug(msg string, f string, content string) {
	log.WithFields(log.Fields{
		"function": f,
		"content":  content,
	}).Debug(msg)
}

/*
// not used
	func logInfo(msg string) {
		log.Info(msg)
	}

	func logWarn(msg string, f string) {
		log.WithFields(log.Fields{
			"function": f,
		}).Warn(msg)
	}
*/

func logError(msg string, f string) {
	log.WithFields(log.Fields{
		"function": f,
	}).Error(msg)
}
