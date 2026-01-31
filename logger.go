package main

import (
	"fmt"
	"os"
	"sync"
	"time"
)

type Logger struct {
	Path string
	file *os.File
	mu   sync.Mutex
}

func (logger *Logger) ensureOpen() error {
	if logger.file != nil {
		return nil
	}
	f, err := os.OpenFile(logger.Path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0640)
	if err != nil {
		fmt.Printf("error opening file: %v. Error: %v\n", logger.Path, err)
		return err
	}
	logger.file = f
	return nil
}

func (logger *Logger) Print(content ...any) {
	logger.mu.Lock()
	defer logger.mu.Unlock()

	if err := logger.ensureOpen(); err != nil {
		return
	}

	stringToLog := fmt.Sprint(content...)
	now := time.Now()
	formatted := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
		now.Year(), now.Month(), now.Day(),
		now.Hour(), now.Minute(), now.Second())
	logString := "[" + formatted + "] " + stringToLog
	logger.file.WriteString(logString)
	logger.file.Sync()
}

func (logger *Logger) Println(content ...any) {
	logger.Print(append(content, "\n")...)
}

func (logger *Logger) Printf(content string, variables ...any) {
	convertedString := fmt.Sprintf(content, variables...)
	logger.Print(convertedString)
}

func (logger *Logger) Close() {
	logger.mu.Lock()
	defer logger.mu.Unlock()

	if logger.file != nil {
		logger.file.Close()
		logger.file = nil
	}
}
