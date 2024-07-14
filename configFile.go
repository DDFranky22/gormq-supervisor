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

func createConfig(configFile string) ConfigFile {
	jsonFile, err := os.Open(configFile)
	if err != nil {
		fmt.Println(err)
	}
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	var configuration ConfigFile

	json.Unmarshal(byteValue, &configuration)

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

	return configuration
}
