package zlog

import (
	"fmt"
	"io"
	"log"
)

func main() {}

func Info(msg ...interface{}) {
	var text string
	text = "[I] "
	for _, txt := range msg {
		text = text + fmt.Sprintf("%v", txt)
	}
	log.Printf("\033[34m%s\033[0m\n", text)
}

func Error(msg ...interface{}) {
	var text string
	text = "[E] "
	for _, txt := range msg {
		text = text + fmt.Sprintf("%v", txt)
	}
	log.Printf("\033[31m%s\033[0m\n", text)
}

func Fatal(msg ...interface{}) {
	var text string
	text = "[F] "
	for _, txt := range msg {
		text = text + fmt.Sprintf("%v", txt)
	}
	log.Fatalf("\033[31m%s\033[0m\n", text)
}

func Success(msg ...interface{}) {
	var text string
	text = "[Success] "
	for _, txt := range msg {
		text = text + fmt.Sprintf("%v", txt)
	}
	log.Printf("\033[32m%s\033[0m\n", text)
}

func PrintText(msg ...interface{}) {
	var text string
	for _, txt := range msg {
		text = text + fmt.Sprintf("%v", txt)
	}
	log.Printf("\033[37m%s\033[0m\n", text)
}

func SetOutput(w io.Writer) {
	log.SetOutput(w)
}
