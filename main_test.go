package main

import (
	"strings"
	"testing"
)

func TestCreateResponse_EmptyCommand(t *testing.T) {
	response := createResponse("")

	if !strings.Contains(response, "Commands available") {
		t.Error("Expected help message for empty command")
	}
}

func TestCreateResponse_WhitespaceOnly(t *testing.T) {
	response := createResponse("   \t\n  ")

	if !strings.Contains(response, "Commands available") {
		t.Error("Expected help message for whitespace-only command")
	}
}

func TestCreateResponse_Status(t *testing.T) {
	// Initialize jobKiller with empty jobs for test
	jobKiller = JobKiller{Jobs: []*Job{}}

	response := createResponse("status")

	// Should return status table header at minimum
	if !strings.Contains(response, "Job") {
		t.Error("Expected status response to contain 'Job' header")
	}
}

func TestCreateResponse_Version(t *testing.T) {
	response := createResponse("version")

	if response != VERSION {
		t.Errorf("Expected version '%s', got '%s'", VERSION, response)
	}
}

func TestCreateResponse_UnknownCommand(t *testing.T) {
	response := createResponse("unknowncommand")

	if !strings.Contains(response, "Commands available") {
		t.Error("Expected help message for unknown command")
	}
}

func TestCreateResponse_StatusOf_NoArgument(t *testing.T) {
	jobKiller = JobKiller{Jobs: []*Job{}}

	response := createResponse("status-of")

	// Should handle missing argument gracefully
	if !strings.Contains(response, "Can't find job") {
		t.Error("Expected 'Can't find job' message for missing argument")
	}
}

func TestCreateResponse_UpdateJob_InsufficientArgs(t *testing.T) {
	response := createResponse("update-job")

	if !strings.Contains(response, "In order to update") {
		t.Error("Expected usage message for insufficient arguments")
	}

	response = createResponse("update-job jobname")

	if !strings.Contains(response, "In order to update") {
		t.Error("Expected usage message for insufficient arguments")
	}

	response = createResponse("update-job jobname property")

	if !strings.Contains(response, "In order to update") {
		t.Error("Expected usage message for insufficient arguments")
	}
}
