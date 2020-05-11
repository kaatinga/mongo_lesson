package logger

import (
	"log"
	"strings"
)

func SubLog(values ...string) {
	var logString string
	logString = strings.Join(values, " ")

	log.Println("└", logString)
}

func SubLogRed(values ...string) {
	var logString string
	logString = strings.Join(values, " ")
	SubLog(logString)
}

func SubSubLogYellow(values ...string) {
	var logString string
	logString = strings.Join(values, " ")
	Subsublog(logString)
}

func SubSubLogRed(values ...string) {
	var logString string
	logString = strings.Join(values, " ")
	Subsublog(logString)
}

func SubSubLogGreen(values ...string) {
	var logString string
	logString = strings.Join(values, " ")
	Subsublog(logString)
}

func Subsublog(values ...string) {
	var logString string
	logString = strings.Join(values, " ")

	log.Println("  └", logString)
}
