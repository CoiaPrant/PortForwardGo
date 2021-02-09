package zlog

import (
	"fmt"
	"io"
	"log"
	"syscall"
)

func main() {}

func Info(msg ...interface{}) {
	var text string
	text = "[I] "
	for _, txt := range msg {
		text = text + fmt.Sprintf("%v", txt)
	}
	WinColorLog(1|8, text)
}

func Error(msg ...interface{}) {
	var text string
	text = "[E] "
	for _, txt := range msg {
		text = text + fmt.Sprintf("%v", txt)
	}
	WinColorLog(4|8, text)
}

func Fatal(msg ...interface{}) {
	var text string
	text = "[F] "
	for _, txt := range msg {
		text = text + fmt.Sprintf("%v", txt)
	}
	WinColorLogFatal(4|8, text)
}

func Success(msg ...interface{}) {
	var text string
	text = "[Success] "
	for _, txt := range msg {
		text = text + fmt.Sprintf("%v", txt)
	}
	WinColorLog(2|8, text)
}

func PrintText(msg ...interface{}) {
	var text string
	for _, txt := range msg {
		text = text + fmt.Sprintf("%v", txt)
	}
	WinColorPrint(7|8, text)
}

func WinColorLog(i int, v ...interface{}) { //设置终端字体颜色
	var s string
	for _, txt := range v {
		s = s + fmt.Sprintf("%v", txt)
	}
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("SetConsoleTextAttribute")
	handle, _, _ := proc.Call(uintptr(syscall.Stdout), uintptr(i))
	log.Printf(s)
	handle, _, _ = proc.Call(uintptr(syscall.Stdout), uintptr(7))
	CloseHandle := kernel32.NewProc("CloseHandle")
	CloseHandle.Call(handle)
}

func WinColorPrint(i int, v ...interface{}) { //设置终端字体颜色
	var s string
	for _, txt := range v {
		s = s + fmt.Sprintf("%v", txt)
	}
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("SetConsoleTextAttribute")
	handle, _, _ := proc.Call(uintptr(syscall.Stdout), uintptr(i))
	fmt.Printf(s)
	handle, _, _ = proc.Call(uintptr(syscall.Stdout), uintptr(7))
	CloseHandle := kernel32.NewProc("CloseHandle")
	CloseHandle.Call(handle)
}

func WinColorLogFatal(i int, v ...interface{}) { //设置终端字体颜色
	var s string
	for _, txt := range v {
		s = s + fmt.Sprintf("%v", txt)
	}
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("SetConsoleTextAttribute")
	handle, _, _ := proc.Call(uintptr(syscall.Stdout), uintptr(i))
	log.Fatalf(s)
	handle, _, _ = proc.Call(uintptr(syscall.Stdout), uintptr(7))
	CloseHandle := kernel32.NewProc("CloseHandle")
	CloseHandle.Call(handle)
}

func SetOutput(w io.Writer) {
	log.SetOutput(w)
}
