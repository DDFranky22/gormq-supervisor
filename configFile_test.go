package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Test getConnectionByName

func TestConfigFile_GetConnectionByName_Found(t *testing.T) {
	config := &ConfigFile{
		ConnectionConfigs: []ConnectionConfig{
			{Name: "conn1", Endpoint: "http://localhost:15672", Username: "user1", Password: "pass1", Vhost: "/"},
			{Name: "conn2", Endpoint: "http://localhost:15673", Username: "user2", Password: "pass2", Vhost: "/test"},
		},
	}

	conn, err := config.getConnectionByName("conn2")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatal("Expected connection to be found")
	}
	if conn.Name != "conn2" {
		t.Errorf("Expected connection name 'conn2', got '%s'", conn.Name)
	}
	if conn.Endpoint != "http://localhost:15673" {
		t.Errorf("Expected endpoint 'http://localhost:15673', got '%s'", conn.Endpoint)
	}
}

func TestConfigFile_GetConnectionByName_NotFound(t *testing.T) {
	config := &ConfigFile{
		ConnectionConfigs: []ConnectionConfig{
			{Name: "conn1", Endpoint: "http://localhost:15672"},
		},
	}

	conn, err := config.getConnectionByName("nonexistent")

	if err == nil {
		t.Error("Expected error for nonexistent connection")
	}
	if conn != nil {
		t.Error("Expected nil connection for nonexistent name")
	}
}

func TestConfigFile_GetConnectionByName_EmptyConnections(t *testing.T) {
	config := &ConfigFile{
		ConnectionConfigs: []ConnectionConfig{},
	}

	conn, err := config.getConnectionByName("conn1")

	if err == nil {
		t.Error("Expected error for empty connections list")
	}
	if conn != nil {
		t.Error("Expected nil connection")
	}
}

func TestConfigFile_GetConnectionByName_FirstMatch(t *testing.T) {
	// If there are duplicates, should return first match
	config := &ConfigFile{
		ConnectionConfigs: []ConnectionConfig{
			{Name: "conn", Endpoint: "http://first:15672"},
			{Name: "conn", Endpoint: "http://second:15672"},
		},
	}

	conn, err := config.getConnectionByName("conn")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if conn.Endpoint != "http://first:15672" {
		t.Errorf("Expected first match endpoint, got '%s'", conn.Endpoint)
	}
}

// Test createConfig

func TestCreateConfig_ValidConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "valid_config.json")

	configContent := `{
		"connections": [
			{
				"name": "main",
				"endpoint": "http://localhost:15672",
				"username": "guest",
				"password": "guest",
				"vhost": "/"
			}
		],
		"jobs": [
			{
				"name": "test_job",
				"groups": ["group1"],
				"sleep_time": 5,
				"sleep_increment": 2,
				"max_sleep": 60,
				"min_messages": 1,
				"command": "echo hello",
				"spawn": 1,
				"connection": "main",
				"queue": "test_queue"
			}
		]
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := createConfig(configPath)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(config.ConnectionConfigs) != 1 {
		t.Errorf("Expected 1 connection, got %d", len(config.ConnectionConfigs))
	}
	if len(config.Jobs) != 1 {
		t.Errorf("Expected 1 job, got %d", len(config.Jobs))
	}
	if config.ConnectionConfigs[0].Name != "main" {
		t.Errorf("Expected connection name 'main', got '%s'", config.ConnectionConfigs[0].Name)
	}
	if config.Jobs[0].Name != "test_job" {
		t.Errorf("Expected job name 'test_job', got '%s'", config.Jobs[0].Name)
	}
}

func TestCreateConfig_WithSpawn(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "spawn_config.json")

	configContent := `{
		"connections": [],
		"jobs": [
			{
				"name": "spawned_job",
				"spawn": 3,
				"command": "echo test",
				"queue": "test_queue"
			}
		]
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := createConfig(configPath)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Original job + 2 clones = 3 total
	if len(config.Jobs) != 3 {
		t.Errorf("Expected 3 jobs after spawn, got %d", len(config.Jobs))
	}

	// Check naming convention
	expectedNames := map[string]bool{
		"spawned_job_0": false,
		"spawned_job_1": false,
		"spawned_job_2": false,
	}

	for _, job := range config.Jobs {
		if _, exists := expectedNames[job.Name]; exists {
			expectedNames[job.Name] = true
		}
	}

	for name, found := range expectedNames {
		if !found {
			t.Errorf("Expected job '%s' not found", name)
		}
	}

	// All jobs should have spawn = 1 after processing
	for _, job := range config.Jobs {
		if job.Spawn != 1 {
			t.Errorf("Expected spawn 1 after processing, got %d for job %s", job.Spawn, job.Name)
		}
	}
}

func TestCreateConfig_EnvVariableReplacement(t *testing.T) {
	os.Setenv("TEST_RABBIT_USER", "envuser")
	os.Setenv("TEST_RABBIT_PASS", "envpass")
	defer os.Unsetenv("TEST_RABBIT_USER")
	defer os.Unsetenv("TEST_RABBIT_PASS")

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "env_config.json")

	configContent := `{
		"connections": [
			{
				"name": "main",
				"endpoint": "http://localhost:15672",
				"username": "${TEST_RABBIT_USER}",
				"password": "${TEST_RABBIT_PASS}",
				"vhost": "/"
			}
		],
		"jobs": []
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := createConfig(configPath)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if config.ConnectionConfigs[0].Username != "envuser" {
		t.Errorf("Expected username 'envuser', got '%s'", config.ConnectionConfigs[0].Username)
	}
	if config.ConnectionConfigs[0].Password != "envpass" {
		t.Errorf("Expected password 'envpass', got '%s'", config.ConnectionConfigs[0].Password)
	}
}

func TestCreateConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid_config.json")

	// Invalid JSON
	configContent := `{ invalid json content }`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err = createConfig(configPath)

	// Should return error for invalid JSON
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestCreateConfig_MissingFile(t *testing.T) {
	_, err := createConfig("/nonexistent/path/config.json")

	if err == nil {
		t.Error("Expected error for missing file")
	}
}

func TestCreateConfig_MultipleSpawnedJobs(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "multi_spawn_config.json")

	configContent := `{
		"connections": [],
		"jobs": [
			{
				"name": "job_a",
				"spawn": 2,
				"command": "echo a",
				"queue": "queue_a"
			},
			{
				"name": "job_b",
				"spawn": 1,
				"command": "echo b",
				"queue": "queue_b"
			}
		]
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := createConfig(configPath)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// job_a spawns 2 (job_a_0, job_a_1), job_b spawns 1 = 3 total
	if len(config.Jobs) != 3 {
		t.Errorf("Expected 3 jobs, got %d", len(config.Jobs))
	}
}

func TestCreateConfig_PreservesJobProperties(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "props_config.json")

	configContent := `{
		"connections": [],
		"jobs": [
			{
				"name": "full_job",
				"groups": ["workers", "batch"],
				"sleep_time": 10,
				"sleep_increment": 5,
				"max_sleep": 120,
				"min_messages": 5,
				"working_dir": "/tmp",
				"user": "worker",
				"command": "php process.php",
				"spawn": 1,
				"connection": "main",
				"queue": "work_queue",
				"error_log_path": "/var/log/",
				"error_log_max_kb_size": 1024,
				"error_log_max_files": 5,
				"max_execution": 300
			}
		]
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := createConfig(configPath)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(config.Jobs) != 1 {
		t.Fatalf("Expected 1 job, got %d", len(config.Jobs))
	}

	job := config.Jobs[0]

	if job.SleepTime != 10 {
		t.Errorf("Expected SleepTime 10, got %d", job.SleepTime)
	}
	if job.SleepIncrement != 5 {
		t.Errorf("Expected SleepIncrement 5, got %d", job.SleepIncrement)
	}
	if job.MaxSleep != 120 {
		t.Errorf("Expected MaxSleep 120, got %d", job.MaxSleep)
	}
	if job.MinMessages != 5 {
		t.Errorf("Expected MinMessages 5, got %d", job.MinMessages)
	}
	if job.WorkingDir != "/tmp" {
		t.Errorf("Expected WorkingDir '/tmp', got '%s'", job.WorkingDir)
	}
	if job.UserId != "worker" {
		t.Errorf("Expected UserId 'worker', got '%s'", job.UserId)
	}
	if job.MaxExecution != 300 {
		t.Errorf("Expected MaxExecution 300, got %d", job.MaxExecution)
	}
	if len(job.Groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(job.Groups))
	}
}
