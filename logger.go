package main

import (
	"fmt"
	"os"
	"time"
)

type Logger struct {
	Path string
}

func (logger *Logger) createFile() *os.File {
	f, err := os.OpenFile(logger.Path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("error opening file: %v. Error: %v\n", logger.Path, err)
	}
	return f
}

func (logger *Logger) Print(content ...any) {
	file := logger.createFile()
	stringToLog := fmt.Sprint(content...)
	now := time.Now()
	formatted := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
		now.Year(), now.Month(), now.Day(),
		now.Hour(), now.Minute(), now.Second())
	logString := "[" + formatted + "] " + stringToLog
	file.WriteString(logString)
	file.Sync()
}

func (logger *Logger) Println(content ...any) {
	logger.Print(append(content, "\n")...)
}

func (logger *Logger) Printf(content string, variables ...any) {
	convertedString := fmt.Sprintf(content, variables...)
	logger.Print(convertedString)
}

func (logger *Logger) Close() {
	file := logger.createFile()
	file.Close()
}
