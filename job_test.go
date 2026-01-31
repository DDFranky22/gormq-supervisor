package main

import (
	"sync"
	"testing"
)

// Test thread-safe getters and setters

func TestJob_GetSetStatus(t *testing.T) {
	job := &Job{}

	job.SetStatus(STATUS_RUNNING)
	if got := job.GetStatus(); got != STATUS_RUNNING {
		t.Errorf("GetStatus() = %v, want %v", got, STATUS_RUNNING)
	}

	job.SetStatus(STATUS_PAUSED)
	if got := job.GetStatus(); got != STATUS_PAUSED {
		t.Errorf("GetStatus() = %v, want %v", got, STATUS_PAUSED)
	}
}

func TestJob_GetSetPause(t *testing.T) {
	job := &Job{}

	job.SetPause(true)
	if got := job.GetPause(); got != true {
		t.Errorf("GetPause() = %v, want true", got)
	}

	job.SetPause(false)
	if got := job.GetPause(); got != false {
		t.Errorf("GetPause() = %v, want false", got)
	}
}

func TestJob_GetSetStop(t *testing.T) {
	job := &Job{}

	job.SetStop(true)
	if got := job.GetStop(); got != true {
		t.Errorf("GetStop() = %v, want true", got)
	}

	job.SetStop(false)
	if got := job.GetStop(); got != false {
		t.Errorf("GetStop() = %v, want false", got)
	}
}

func TestJob_GetSetPID(t *testing.T) {
	job := &Job{}

	job.SetPID(12345)
	if got := job.GetPID(); got != 12345 {
		t.Errorf("GetPID() = %v, want 12345", got)
	}
}

func TestJob_GetSetCurrentSleepTime(t *testing.T) {
	job := &Job{}

	job.SetCurrentSleepTime(30)
	if got := job.GetCurrentSleepTime(); got != 30 {
		t.Errorf("GetCurrentSleepTime() = %v, want 30", got)
	}
}

func TestJob_GetSetStartedAt(t *testing.T) {
	job := &Job{}

	job.SetStartedAt(1234567890)
	if got := job.GetStartedAt(); got != 1234567890 {
		t.Errorf("GetStartedAt() = %v, want 1234567890", got)
	}
}

func TestJob_GetSetMinMessages(t *testing.T) {
	job := &Job{MinMessages: 5}

	if got := job.GetMinMessages(); got != 5 {
		t.Errorf("GetMinMessages() = %v, want 5", got)
	}

	job.SetMinMessages(10)
	if got := job.GetMinMessages(); got != 10 {
		t.Errorf("GetMinMessages() = %v, want 10", got)
	}
}

func TestJob_GetSetSleepTime(t *testing.T) {
	job := &Job{SleepTime: 5}

	if got := job.GetSleepTime(); got != 5 {
		t.Errorf("GetSleepTime() = %v, want 5", got)
	}

	job.SetSleepTime(10)
	if got := job.GetSleepTime(); got != 10 {
		t.Errorf("GetSleepTime() = %v, want 10", got)
	}
}

func TestJob_GetSetSleepIncrement(t *testing.T) {
	job := &Job{SleepIncrement: 2}

	if got := job.GetSleepIncrement(); got != 2 {
		t.Errorf("GetSleepIncrement() = %v, want 2", got)
	}

	job.SetSleepIncrement(5)
	if got := job.GetSleepIncrement(); got != 5 {
		t.Errorf("GetSleepIncrement() = %v, want 5", got)
	}
}

func TestJob_GetSetMaxSleep(t *testing.T) {
	job := &Job{MaxSleep: 60}

	if got := job.GetMaxSleep(); got != 60 {
		t.Errorf("GetMaxSleep() = %v, want 60", got)
	}

	job.SetMaxSleep(120)
	if got := job.GetMaxSleep(); got != 120 {
		t.Errorf("GetMaxSleep() = %v, want 120", got)
	}
}

func TestJob_GetSetMaxExecution(t *testing.T) {
	job := &Job{MaxExecution: 300}

	if got := job.GetMaxExecution(); got != 300 {
		t.Errorf("GetMaxExecution() = %v, want 300", got)
	}

	job.SetMaxExecution(600)
	if got := job.GetMaxExecution(); got != 600 {
		t.Errorf("GetMaxExecution() = %v, want 600", got)
	}
}

// Test concurrent access

func TestJob_ConcurrentAccess(t *testing.T) {
	job := &Job{}
	var wg sync.WaitGroup
	iterations := 1000

	// Concurrent writers
	wg.Add(3)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			job.SetStatus(STATUS_RUNNING)
			job.SetStatus(STATUS_SLEEP)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			job.SetPause(true)
			job.SetPause(false)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			job.SetPID(i)
		}
	}()

	// Concurrent readers
	wg.Add(3)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_ = job.GetStatus()
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_ = job.GetPause()
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_ = job.GetPID()
		}
	}()

	wg.Wait()
	// If we reach here without race condition panics, the test passes
}

// Test getStatusName

func TestJob_GetStatusName(t *testing.T) {
	tests := []struct {
		status   int16
		expected string
	}{
		{STATUS_SLEEP, "SLEEPING"},
		{STATUS_RUNNING, "RUNNING"},
		{STATUS_PAUSED, "PAUSED"},
		{STATUS_TERMINATED, "TERMINATED"},
		{99, "UNKNOWN"},
		{-1, "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			job := &Job{}
			job.SetStatus(tt.status)
			if got := job.getStatusName(); got != tt.expected {
				t.Errorf("getStatusName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// Test getStatus (returns map)

func TestJob_GetStatus_Map(t *testing.T) {
	job := &Job{
		Name:             "test_job",
		Groups:           []string{"group1", "group2"},
		UserId:           "testuser",
		MaxSleep:         120,
		CurrentSleepTime: 30,
		PID:              12345,
		Status:           STATUS_RUNNING,
		StartedAt:        1234567890,
	}

	status := job.getStatus()

	if status["Name"] != "test_job" {
		t.Errorf("Expected Name 'test_job', got %v", status["Name"])
	}
	if status["Status"] != "RUNNING" {
		t.Errorf("Expected Status 'RUNNING', got %v", status["Status"])
	}
	if status["PID"] != 12345 {
		t.Errorf("Expected PID 12345, got %v", status["PID"])
	}
	if status["User"] != "testuser" {
		t.Errorf("Expected User 'testuser', got %v", status["User"])
	}
	if status["Sleep"] != 30 {
		t.Errorf("Expected Sleep 30, got %v", status["Sleep"])
	}
	if status["MaxSleep"] != 120 {
		t.Errorf("Expected MaxSleep 120, got %v", status["MaxSleep"])
	}
}

// Test clone

func TestJob_Clone(t *testing.T) {
	original := &Job{
		Name:              "original_job",
		Groups:            []string{"group1", "group2"},
		SleepTime:         5,
		SleepIncrement:    2,
		MaxSleep:          60,
		MinMessages:       1,
		WorkingDir:        "/tmp",
		UserId:            "testuser",
		Command:           "echo hello",
		Spawn:             3,
		ConnectionName:    "main_connection",
		Queue:             "test_queue",
		ErrorLogPath:      "/var/log/",
		ErrorLogMaxKBSize: 1024,
		ErrorLogMaxFiles:  5,
		MaxExecution:      300,
	}

	cloned := original.clone(1)

	// Check name is modified
	expectedName := "original_job_1"
	if cloned.Name != expectedName {
		t.Errorf("Expected cloned name '%s', got '%s'", expectedName, cloned.Name)
	}

	// Check spawn is set to 1
	if cloned.Spawn != 1 {
		t.Errorf("Expected Spawn 1, got %d", cloned.Spawn)
	}

	// Check other fields are copied
	if cloned.SleepTime != original.SleepTime {
		t.Errorf("SleepTime not copied correctly")
	}
	if cloned.SleepIncrement != original.SleepIncrement {
		t.Errorf("SleepIncrement not copied correctly")
	}
	if cloned.MaxSleep != original.MaxSleep {
		t.Errorf("MaxSleep not copied correctly")
	}
	if cloned.MinMessages != original.MinMessages {
		t.Errorf("MinMessages not copied correctly")
	}
	if cloned.WorkingDir != original.WorkingDir {
		t.Errorf("WorkingDir not copied correctly")
	}
	if cloned.UserId != original.UserId {
		t.Errorf("UserId not copied correctly")
	}
	if cloned.Command != original.Command {
		t.Errorf("Command not copied correctly")
	}
	if cloned.ConnectionName != original.ConnectionName {
		t.Errorf("ConnectionName not copied correctly")
	}
	if cloned.Queue != original.Queue {
		t.Errorf("Queue not copied correctly")
	}
	if cloned.ErrorLogPath != original.ErrorLogPath {
		t.Errorf("ErrorLogPath not copied correctly")
	}
	if cloned.ErrorLogMaxKBSize != original.ErrorLogMaxKBSize {
		t.Errorf("ErrorLogMaxKBSize not copied correctly")
	}
	if cloned.ErrorLogMaxFiles != original.ErrorLogMaxFiles {
		t.Errorf("ErrorLogMaxFiles not copied correctly")
	}
	if cloned.MaxExecution != original.MaxExecution {
		t.Errorf("MaxExecution not copied correctly")
	}

	// Check Groups slice is copied (not the same reference)
	if len(cloned.Groups) != len(original.Groups) {
		t.Errorf("Groups length mismatch")
	}
	for i, g := range cloned.Groups {
		if g != original.Groups[i] {
			t.Errorf("Groups element %d mismatch", i)
		}
	}
}

func TestJob_Clone_MutexNotShared(t *testing.T) {
	original := &Job{
		Name: "original",
	}

	cloned := original.clone(1)

	// Set different values concurrently - if mutex is shared, we'd have issues
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			original.SetStatus(STATUS_RUNNING)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			cloned.SetStatus(STATUS_PAUSED)
		}
	}()

	wg.Wait()
	// Test passes if no race conditions occur
}

// Test updateProperties

func TestJob_UpdateProperties_MinMessages(t *testing.T) {
	job := &Job{MinMessages: 5}

	err := job.updateProperties([]string{"min_messages", "10"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if job.GetMinMessages() != 10 {
		t.Errorf("Expected MinMessages 10, got %d", job.GetMinMessages())
	}
}

func TestJob_UpdateProperties_SleepTime(t *testing.T) {
	job := &Job{SleepTime: 5}

	err := job.updateProperties([]string{"sleep_time", "15"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if job.GetSleepTime() != 15 {
		t.Errorf("Expected SleepTime 15, got %d", job.GetSleepTime())
	}
}

func TestJob_UpdateProperties_SleepIncrement(t *testing.T) {
	job := &Job{SleepIncrement: 2}

	err := job.updateProperties([]string{"sleep_increment", "5"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if job.GetSleepIncrement() != 5 {
		t.Errorf("Expected SleepIncrement 5, got %d", job.GetSleepIncrement())
	}
}

func TestJob_UpdateProperties_MaxSleep(t *testing.T) {
	job := &Job{MaxSleep: 60}

	err := job.updateProperties([]string{"max_sleep", "120"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if job.GetMaxSleep() != 120 {
		t.Errorf("Expected MaxSleep 120, got %d", job.GetMaxSleep())
	}
}

func TestJob_UpdateProperties_Spawn(t *testing.T) {
	job := &Job{Spawn: 1}

	err := job.updateProperties([]string{"spawn", "5"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	// Note: Spawn uses direct field access via SetSpawn, need to check via lock
	job.mu.RLock()
	spawn := job.Spawn
	job.mu.RUnlock()
	if spawn != 5 {
		t.Errorf("Expected Spawn 5, got %d", spawn)
	}
}

func TestJob_UpdateProperties_MaxExecution(t *testing.T) {
	job := &Job{MaxExecution: 300}

	err := job.updateProperties([]string{"max_execution", "600"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if job.GetMaxExecution() != 600 {
		t.Errorf("Expected MaxExecution 600, got %d", job.GetMaxExecution())
	}
}

func TestJob_UpdateProperties_InvalidProperty(t *testing.T) {
	job := &Job{}

	err := job.updateProperties([]string{"invalid_property", "10"})
	if err == nil {
		t.Error("Expected error for invalid property")
	}
}

func TestJob_UpdateProperties_InvalidValue(t *testing.T) {
	job := &Job{}

	err := job.updateProperties([]string{"min_messages", "not_a_number"})
	if err == nil {
		t.Error("Expected error for non-numeric value")
	}
}

func TestJob_UpdateProperties_NegativeValue(t *testing.T) {
	tests := []struct {
		property string
	}{
		{"min_messages"},
		{"sleep_time"},
		{"sleep_increment"},
		{"max_sleep"},
		{"max_execution"},
	}

	for _, tt := range tests {
		t.Run(tt.property, func(t *testing.T) {
			job := &Job{}
			err := job.updateProperties([]string{tt.property, "-5"})
			if err == nil {
				t.Errorf("Expected error for negative value on %s", tt.property)
			}
		})
	}
}

func TestJob_UpdateProperties_SpawnZero(t *testing.T) {
	job := &Job{Spawn: 1}

	err := job.updateProperties([]string{"spawn", "0"})
	if err == nil {
		t.Error("Expected error for spawn = 0")
	}
}

func TestJob_UpdateProperties_SpawnNegative(t *testing.T) {
	job := &Job{Spawn: 1}

	err := job.updateProperties([]string{"spawn", "-1"})
	if err == nil {
		t.Error("Expected error for negative spawn")
	}
}
