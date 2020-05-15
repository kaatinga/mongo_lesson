package logger

import (
	"github.com/fatih/color"
	"io"
	"log"
)

var (
	Info       *log.Logger
	SubInfo    *log.Logger
	SubSubInfo *log.Logger
)

func LoggerInit(infoHandle io.Writer) {

	log.SetOutput(infoHandle)

	Info = log.New(infoHandle,
		"", 0)

	SubInfo = log.New(infoHandle,
		" └ ", 0)

	SubSubInfo = log.New(infoHandle,
		"   └ ", 0)
}

func Yellow(logger *log.Logger, text string) {
	color.Set(color.FgHiYellow)
	logger.Println(text)
	color.Unset()
}

func Green(logger *log.Logger, text string) {
	color.Set(color.FgHiGreen)
	logger.Println(text)
	color.Unset()
}

func Red(logger *log.Logger, text string) {
	color.Set(color.FgHiRed)
	logger.Println(text)
	color.Unset()
}
