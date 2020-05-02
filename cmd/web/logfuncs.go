package main

import (
	"github.com/fatih/color"
	"log"
	"strings"
)

func sublog(values ...string) {
	var logString string
	logString = strings.Join(values, " ")

	log.Println("└", logString)
}

func subLogRed(values ...string) {
	var logString string
	logString = strings.Join(values, " ")
	color.Set(color.FgHiRed)
	sublog(logString)
	color.Unset()
}

func subSubLogYellow(values ...string) {
	var logString string
	logString = strings.Join(values, " ")
	color.Set(color.FgHiYellow)
	subsublog(logString)
	color.Unset()
}

func subSubLogRed(values ...string) {
	var logString string
	logString = strings.Join(values, " ")
	color.Set(color.FgHiRed)
	subsublog(logString)
	color.Unset()
}

func subSubLogGreen(values ...string) {
	var logString string
	logString = strings.Join(values, " ")
	color.Set(color.FgHiGreen)
	subsublog(logString)
	color.Unset()
}

func subsublog(values ...string) {
	var logString string
	logString = strings.Join(values, " ")

	log.Println("  └", logString)
}
