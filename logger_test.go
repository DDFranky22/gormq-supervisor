package main

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestLogger_Print(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger := Logger{Path: logPath}
	defer logger.Close()

	logger.Print("test message")

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "test message") {
		t.Errorf("Expected log to contain 'test message', got: %s", content)
	}
}

func TestLogger_Println(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger := Logger{Path: logPath}
	defer logger.Close()

	logger.Println("line 1")
	logger.Println("line 2")

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "line 1") {
		t.Error("Expected log to contain 'line 1'")
	}
	if !strings.Contains(string(content), "line 2") {
		t.Error("Expected log to contain 'line 2'")
	}
}

func TestLogger_Printf(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger := Logger{Path: logPath}
	defer logger.Close()

	logger.Printf("value: %d, name: %s", 42, "test")

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "value: 42, name: test") {
		t.Errorf("Expected formatted output, got: %s", content)
	}
}

func TestLogger_Close(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger := Logger{Path: logPath}

	logger.Println("before close")
	logger.Close()

	// Should be able to close multiple times without panic
	logger.Close()

	// Should be able to write after close (reopens file)
	logger.Println("after close")
	logger.Close()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "after close") {
		t.Error("Expected log to contain 'after close' after reopen")
	}
}

func TestLogger_ConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger := Logger{Path: logPath}
	defer logger.Close()

	var wg sync.WaitGroup
	iterations := 100

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				logger.Printf("goroutine %d, iteration %d\n", id, j)
			}
		}(i)
	}

	wg.Wait()

	// Verify file exists and has content
	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("Failed to stat log file: %v", err)
	}
	if info.Size() == 0 {
		t.Error("Expected log file to have content")
	}
}

func TestLogger_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger := Logger{Path: logPath}
	logger.Println("test")
	logger.Close()

	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("Failed to stat log file: %v", err)
	}

	// Check that file is not world-writable (mode should be 0640 or stricter)
	mode := info.Mode().Perm()
	if mode&0002 != 0 {
		t.Errorf("Log file should not be world-writable, got mode: %o", mode)
	}
	if mode&0020 != 0 {
		t.Errorf("Log file should not be group-writable, got mode: %o", mode)
	}
}

func TestLogger_InvalidPath(t *testing.T) {
	// Try to create logger with invalid path
	logger := Logger{Path: "/nonexistent/directory/test.log"}

	// Should not panic, just fail silently
	logger.Println("test")
	logger.Close()
}

func TestLogger_TimestampFormat(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger := Logger{Path: logPath}
	defer logger.Close()

	logger.Println("test")

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Check timestamp format: [YYYY-MM-DDTHH:MM:SS]
	if !strings.Contains(string(content), "[") || !strings.Contains(string(content), "T") {
		t.Errorf("Expected timestamp in log, got: %s", content)
	}
}
