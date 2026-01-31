// main package
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

type ConfigFile struct {
	ConnectionConfigs []ConnectionConfig `json:"connections"`
	Jobs              []Job              `json:"jobs"`
}

func (configFile *ConfigFile) getConnectionByName(name string) (*ConnectionConfig, error) {
	for k := 0; k < len(configFile.ConnectionConfigs); k++ {
		if name == configFile.ConnectionConfigs[k].Name {
			return &configFile.ConnectionConfigs[k], nil
		}
	}
	return nil, errors.New("missing connection in config")
}

func createConfig(configFile string) (ConfigFile, error) {
	var configuration ConfigFile

	jsonFile, err := os.Open(configFile)
	if err != nil {
		return configuration, fmt.Errorf("failed to open config file %s: %w", configFile, err)
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return configuration, fmt.Errorf("failed to read config file %s: %w", configFile, err)
	}

	if err := json.Unmarshal(byteValue, &configuration); err != nil {
		return configuration, fmt.Errorf("failed to parse config file %s: %w", configFile, err)
	}

	for index := 0; index < len(configuration.ConnectionConfigs); index++ {
		configuration.ConnectionConfigs[index].replaceEnvVariables()
	}

	for job := 0; job < len(configuration.Jobs); job++ {
		if configuration.Jobs[job].Spawn > 1 {
			for spawn := 1; spawn < configuration.Jobs[job].Spawn; spawn++ {
				newClonedJob := configuration.Jobs[job].clone(spawn)
				configuration.Jobs = append(configuration.Jobs, newClonedJob)
			}
			configuration.Jobs[job].Name = configuration.Jobs[job].Name + "_0"
			configuration.Jobs[job].Spawn = 1
		}
	}

	return configuration, nil
}
