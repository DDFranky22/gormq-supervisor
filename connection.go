package main

import (
	"os"
	"strings"
)

type ConnectionConfig struct {
	Name     string `json:"name"`
	Endpoint string `json:"endpoint"`
	Username string `json:"username"`
	Password string `json:"password"`
	Vhost    string `json:"vhost"`
}

func (connectionConfig *ConnectionConfig) replaceEnvVariables() *ConnectionConfig {
	connectionConfig.Username = replaceEnvVar(connectionConfig.Username)
	connectionConfig.Password = replaceEnvVar(connectionConfig.Password)
	return connectionConfig
}

func replaceEnvVar(envVarName string) string {
	if strings.HasPrefix(envVarName, "${") && strings.HasSuffix(envVarName, "}") {
		var trimmedValue string = strings.TrimLeft(strings.TrimRight(envVarName, "}"), "${")
		return os.Getenv(trimmedValue)
	}
	return envVarName
}
