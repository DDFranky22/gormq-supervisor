package main

import (
	"context"
	"strings"
	"testing"
)

func createTestJob(name string, groups []string) *Job {
	ctx, cancel := context.WithCancel(context.Background())
	return &Job{
		Name:             name,
		Groups:           groups,
		Status:           STATUS_SLEEP,
		Pause:            false,
		Stop:             false,
		OwnContext:       ctx,
		OwnContextCancel: cancel,
	}
}

// Test pause/unpause single job

func TestJobKiller_Pause(t *testing.T) {
	job1 := createTestJob("job1", nil)
	job2 := createTestJob("job2", nil)

	jk := &JobKiller{Jobs: []*Job{job1, job2}}

	jk.pause("job1")

	if !job1.GetPause() {
		t.Error("Expected job1 to be paused")
	}
	if job2.GetPause() {
		t.Error("Expected job2 to remain unpaused")
	}
}

func TestJobKiller_Pause_NonExistent(t *testing.T) {
	job1 := createTestJob("job1", nil)

	jk := &JobKiller{Jobs: []*Job{job1}}

	jk.pause("nonexistent")

	if job1.GetPause() {
		t.Error("Expected job1 to remain unpaused")
	}
}

func TestJobKiller_Pause_AlreadyPaused(t *testing.T) {
	job1 := createTestJob("job1", nil)
	job1.SetPause(true)

	jk := &JobKiller{Jobs: []*Job{job1}}

	jk.pause("job1")

	if !job1.GetPause() {
		t.Error("Expected job1 to still be paused")
	}
}

func TestJobKiller_Unpause(t *testing.T) {
	job1 := createTestJob("job1", nil)
	job1.SetPause(true)
	job2 := createTestJob("job2", nil)
	job2.SetPause(true)

	jk := &JobKiller{Jobs: []*Job{job1, job2}}

	jk.unpause("job1")

	if job1.GetPause() {
		t.Error("Expected job1 to be unpaused")
	}
	if !job2.GetPause() {
		t.Error("Expected job2 to remain paused")
	}
}

func TestJobKiller_Unpause_NotPaused(t *testing.T) {
	job1 := createTestJob("job1", nil)

	jk := &JobKiller{Jobs: []*Job{job1}}

	jk.unpause("job1")

	if job1.GetPause() {
		t.Error("Expected job1 to remain unpaused")
	}
}

// Test pauseAll/unpauseAll

func TestJobKiller_PauseAll(t *testing.T) {
	job1 := createTestJob("job1", nil)
	job2 := createTestJob("job2", nil)
	job3 := createTestJob("job3", nil)

	jk := &JobKiller{Jobs: []*Job{job1, job2, job3}}

	jk.pauseAll()

	if !job1.GetPause() {
		t.Error("Expected job1 to be paused")
	}
	if !job2.GetPause() {
		t.Error("Expected job2 to be paused")
	}
	if !job3.GetPause() {
		t.Error("Expected job3 to be paused")
	}
}

func TestJobKiller_PauseAll_SomeAlreadyPaused(t *testing.T) {
	job1 := createTestJob("job1", nil)
	job1.SetPause(true)
	job2 := createTestJob("job2", nil)

	jk := &JobKiller{Jobs: []*Job{job1, job2}}

	jk.pauseAll()

	if !job1.GetPause() {
		t.Error("Expected job1 to still be paused")
	}
	if !job2.GetPause() {
		t.Error("Expected job2 to be paused")
	}
}

func TestJobKiller_UnpauseAll(t *testing.T) {
	job1 := createTestJob("job1", nil)
	job1.SetPause(true)
	job2 := createTestJob("job2", nil)
	job2.SetPause(true)
	job3 := createTestJob("job3", nil)
	job3.SetPause(true)

	jk := &JobKiller{Jobs: []*Job{job1, job2, job3}}

	jk.unpauseAll()

	if job1.GetPause() {
		t.Error("Expected job1 to be unpaused")
	}
	if job2.GetPause() {
		t.Error("Expected job2 to be unpaused")
	}
	if job3.GetPause() {
		t.Error("Expected job3 to be unpaused")
	}
}

func TestJobKiller_UnpauseAll_SomeNotPaused(t *testing.T) {
	job1 := createTestJob("job1", nil)
	job1.SetPause(true)
	job2 := createTestJob("job2", nil)

	jk := &JobKiller{Jobs: []*Job{job1, job2}}

	jk.unpauseAll()

	if job1.GetPause() {
		t.Error("Expected job1 to be unpaused")
	}
	if job2.GetPause() {
		t.Error("Expected job2 to remain unpaused")
	}
}

// Test pauseGroup/unpauseGroup

func TestJobKiller_PauseGroup(t *testing.T) {
	job1 := createTestJob("job1", []string{"groupA", "groupB"})
	job2 := createTestJob("job2", []string{"groupA"})
	job3 := createTestJob("job3", []string{"groupC"})

	jk := &JobKiller{Jobs: []*Job{job1, job2, job3}}

	jk.pauseGroup("groupA")

	if !job1.GetPause() {
		t.Error("Expected job1 (groupA member) to be paused")
	}
	if !job2.GetPause() {
		t.Error("Expected job2 (groupA member) to be paused")
	}
	if job3.GetPause() {
		t.Error("Expected job3 (groupC only) to remain unpaused")
	}
}

func TestJobKiller_PauseGroup_NonExistent(t *testing.T) {
	job1 := createTestJob("job1", []string{"groupA"})

	jk := &JobKiller{Jobs: []*Job{job1}}

	jk.pauseGroup("nonexistent")

	if job1.GetPause() {
		t.Error("Expected job1 to remain unpaused")
	}
}

func TestJobKiller_UnpauseGroup(t *testing.T) {
	job1 := createTestJob("job1", []string{"groupA", "groupB"})
	job1.SetPause(true)
	job2 := createTestJob("job2", []string{"groupA"})
	job2.SetPause(true)
	job3 := createTestJob("job3", []string{"groupC"})
	job3.SetPause(true)

	jk := &JobKiller{Jobs: []*Job{job1, job2, job3}}

	jk.unpauseGroup("groupA")

	if job1.GetPause() {
		t.Error("Expected job1 (groupA member) to be unpaused")
	}
	if job2.GetPause() {
		t.Error("Expected job2 (groupA member) to be unpaused")
	}
	if !job3.GetPause() {
		t.Error("Expected job3 (groupC only) to remain paused")
	}
}

func TestJobKiller_UnpauseGroup_EmptyGroups(t *testing.T) {
	job1 := createTestJob("job1", nil)
	job1.SetPause(true)

	jk := &JobKiller{Jobs: []*Job{job1}}

	jk.unpauseGroup("groupA")

	if !job1.GetPause() {
		t.Error("Expected job1 (no groups) to remain paused")
	}
}

// Test findJobByName

func TestJobKiller_FindJobByName_Found(t *testing.T) {
	job1 := createTestJob("job1", nil)
	job2 := createTestJob("job2", nil)

	jk := &JobKiller{Jobs: []*Job{job1, job2}}

	found, err := jk.findJobByName("job2")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if found != job2 {
		t.Error("Expected to find job2")
	}
}

func TestJobKiller_FindJobByName_NotFound(t *testing.T) {
	job1 := createTestJob("job1", nil)

	jk := &JobKiller{Jobs: []*Job{job1}}

	found, err := jk.findJobByName("nonexistent")

	if err == nil {
		t.Error("Expected error for nonexistent job")
	}
	if found != nil {
		t.Error("Expected nil job for nonexistent name")
	}
}

func TestJobKiller_FindJobByName_EmptyJobs(t *testing.T) {
	jk := &JobKiller{Jobs: []*Job{}}

	found, err := jk.findJobByName("job1")

	if err == nil {
		t.Error("Expected error for empty jobs list")
	}
	if found != nil {
		t.Error("Expected nil job")
	}
}

// Test returnStatus

func TestJobKiller_ReturnStatus(t *testing.T) {
	job1 := createTestJob("job1", []string{"groupA"})
	job1.SetStatus(STATUS_RUNNING)
	job1.SetPID(12345)
	job1.UserId = "testuser"

	job2 := createTestJob("job2", []string{"groupB"})
	job2.SetStatus(STATUS_PAUSED)

	jk := &JobKiller{Jobs: []*Job{job1, job2}}

	status := jk.returnStatus()

	// Check header is present
	if !strings.Contains(status, "Job") {
		t.Error("Expected header to contain 'Job'")
	}
	if !strings.Contains(status, "Status") {
		t.Error("Expected header to contain 'Status'")
	}
	if !strings.Contains(status, "PID") {
		t.Error("Expected header to contain 'PID'")
	}

	// Check job data is present
	if !strings.Contains(status, "job1") {
		t.Error("Expected status to contain 'job1'")
	}
	if !strings.Contains(status, "job2") {
		t.Error("Expected status to contain 'job2'")
	}
	if !strings.Contains(status, "RUNNING") {
		t.Error("Expected status to contain 'RUNNING'")
	}
	if !strings.Contains(status, "PAUSED") {
		t.Error("Expected status to contain 'PAUSED'")
	}
}

func TestJobKiller_ReturnStatus_EmptyJobs(t *testing.T) {
	jk := &JobKiller{Jobs: []*Job{}}

	status := jk.returnStatus()

	// Should still have header
	if !strings.Contains(status, "Job") {
		t.Error("Expected header to be present even with no jobs")
	}
}

// Test returnStatusOf

func TestJobKiller_ReturnStatusOf_Found(t *testing.T) {
	job1 := createTestJob("job1", []string{"groupA"})
	job1.SetStatus(STATUS_RUNNING)
	job1.SetPID(12345)
	job1.UserId = "testuser"
	job1.MaxSleep = 120

	job2 := createTestJob("job2", nil)

	jk := &JobKiller{Jobs: []*Job{job1, job2}}

	status := jk.returnStatusOf("job1")

	if !strings.Contains(status, "job1") {
		t.Error("Expected status to contain 'job1'")
	}
	if !strings.Contains(status, "RUNNING") {
		t.Error("Expected status to contain 'RUNNING'")
	}
	if !strings.Contains(status, "Max sleep") {
		t.Error("Expected status to contain 'Max sleep' header")
	}
	// Should not contain job2
	if strings.Contains(status, "job2") {
		t.Error("Expected status to not contain 'job2'")
	}
}

func TestJobKiller_ReturnStatusOf_NotFound(t *testing.T) {
	job1 := createTestJob("job1", nil)

	jk := &JobKiller{Jobs: []*Job{job1}}

	status := jk.returnStatusOf("nonexistent")

	if !strings.Contains(status, "Can't find job") {
		t.Error("Expected 'Can't find job' message")
	}
	if !strings.Contains(status, "nonexistent") {
		t.Error("Expected job name in error message")
	}
}

// Test killAll

func TestJobKiller_KillAll(t *testing.T) {
	job1 := createTestJob("job1", nil)
	job1.SetStatus(STATUS_RUNNING)

	job2 := createTestJob("job2", nil)
	job2.SetStatus(STATUS_RUNNING)

	jk := &JobKiller{Jobs: []*Job{job1, job2}}

	jk.killAll()

	// Check stop flag is set
	if !job1.GetStop() {
		t.Error("Expected job1 Stop to be true")
	}
	if !job2.GetStop() {
		t.Error("Expected job2 Stop to be true")
	}

	// Check status is TERMINATED
	if job1.GetStatus() != STATUS_TERMINATED {
		t.Errorf("Expected job1 status TERMINATED, got %d", job1.GetStatus())
	}
	if job2.GetStatus() != STATUS_TERMINATED {
		t.Errorf("Expected job2 status TERMINATED, got %d", job2.GetStatus())
	}

	// Check context was cancelled (by checking if context.Err() returns non-nil)
	if job1.OwnContext.Err() == nil {
		t.Error("Expected job1 context to be cancelled")
	}
	if job2.OwnContext.Err() == nil {
		t.Error("Expected job2 context to be cancelled")
	}
}

func TestJobKiller_KillAll_WithNilCmd(t *testing.T) {
	job1 := createTestJob("job1", nil)
	job1.SetPID(12345)
	job1.SetCmdExecutable(nil) // No command

	jk := &JobKiller{Jobs: []*Job{job1}}

	// Should not panic
	jk.killAll()

	if !job1.GetStop() {
		t.Error("Expected job1 Stop to be true")
	}
}

func TestJobKiller_KillAll_EmptyJobs(t *testing.T) {
	jk := &JobKiller{Jobs: []*Job{}}

	// Should not panic
	jk.killAll()
}
