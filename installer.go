package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

var (
	pathFile               = "/etc/systemd/system/gormq-supervisor.service"
	initDFile              = "/etc/init.d/gormq-supervisor"
	serviceName            = "gormq-supervisor"
	description            = "Service written in GO to check on RabbitMQ before launching commands for consumers"
	defaultUser            = "root"
	defaultGroup           = "root"
	defaultEnvironmentFile = "./gormq.env"
)

func StringPrompt(label string) string {
	var s string
	r := bufio.NewReader(os.Stdin)
	for {
		fmt.Fprint(os.Stderr, label+" ")
		s, _ = r.ReadString('\n')
		if s != "" {
			break
		}
	}
	return strings.TrimSpace(s)
}

func installAsServicectl() {
	//create file under /etc/systemd/system/gormq-supervisor.service
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		return
	}

	currentExec, err := os.Executable()
	if err != nil {
		fmt.Println(err)
		return
	}

	if _, err := os.Stat(pathFile); err == nil {
		fmt.Printf("Service file at %v already exists. Service might be already installed.\n", pathFile)
		return
	} else if errors.Is(err, os.ErrNotExist) {

		user := defaultUser
		group := defaultGroup
		environmentFile := defaultEnvironmentFile

		if !*silentInstall {
			user = StringPrompt("Inser the user that should launch the service:")
			group = StringPrompt("Insert the group of the user:")
			environmentFile = StringPrompt("Indicate the path of the EnvironmentFile (absolute):")
		}

		serviceFile, err := os.OpenFile(pathFile, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Println(err)
		}
		defer serviceFile.Close()

		fileContent := fmt.Sprintf(
			`[Unit]
Description=%s
ConditionPathExists=%s
After=network.target
[Service]
Type=simple
User=%s
Group=%s
WorkingDirectory=%s
EnvironmentFile=%s
ExecStart=%s -config $CONFIG_FILE_PATH -log $LOG_PATH -port $LISTENING_PORT
Restart=on-failure
RestartSec=10
StandardOutput=syslog
StandardError=syslog
SyslogIdentifier=gormqsupervisorservice
[Install]
WantedBy=multi-user.target`, description, currentDir, user, group, currentDir, environmentFile, currentExec)

		serviceFile.WriteString(fileContent)
		serviceFile.Sync()

		fmt.Println("Service installed")
		return
	} else {
		fmt.Println("Something is not right. Terminating")
		return
	}
}

func installAsInitd() {
	//create file under /etc/init.d/gormq-supervisor
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		return
	}

	if _, err := os.Stat(initDFile); err == nil {
		fmt.Printf("Service file at %v already exists. Service might be already installed.\n", initDFile)
		return
	} else if errors.Is(err, os.ErrNotExist) {

		chkconfig := "35 95 05"
		logFile := currentDir + "/"
		configFile := currentDir + "/cong-configuration.json"
		listeningPort := "9000"

		if !*silentInstall {

			chkconfig = StringPrompt("chkconfig (35 95 05)")
			if chkconfig == "" {
				chkconfig = "35 95 05"
			}

			logFile = StringPrompt("where does the app logs? (./)")
			if logFile == "" {
				logFile = currentDir + "/"
			}

			configFile = StringPrompt("Configuration file (./gormq-configuration.json)")
			if configFile == "" {
				configFile = currentDir + "/gormq-configuration.json"
			}

			listeningPort = StringPrompt("On what port should the service listen? (9000)")
			if listeningPort == "" {
				listeningPort = "9000"
			}
		}

		serviceFile, err := os.OpenFile(initDFile, os.O_CREATE|os.O_WRONLY, 0755)
		if err != nil {
			fmt.Println(err)
		}
		defer serviceFile.Close()

		fileContent :=
			`#!/bin/bash
#
# chkconfig: %s
# description: %s.
### BEGIN INIT INFO
# Provides: %s
# Required-Start:	$rabbitmq-server
# Required-Stop:	$rabbitmq-server
# Default-Start:
# Default-Stop:
# Short-Description: %s
### END INIT INFO
# Load functions from library
. /etc/init.d/functions

# Name of the application
app=%s
workingDirectory=%s
configFile=%s
logFile=%s
listeningPort=%s

# Start the service
run() {
	echo -n $"Starting $app:"
	cd $workingDirectory
	./$app --config $configFile --log $logFile --port $listeningPort > /var/log/$app.log 2> /var/log/$app.err < /dev/null &

	sleep 1

	status $app > /dev/null
	# If application is running
	if [[ $? -eq 0 ]]; then
		# Store PID in lock file
		echo $! > /var/lock/subsys/$app
		success
		echo
	else
		failure
		echo
	fi
}

# Start the service
start() {
	status $app > /dev/null
	# If application is running
	if [[ $? -eq 0 ]]; then
		status $app
	else
		run
	fi
}

# Restart the service
stop() {
	echo -n "Stopping $app: "
	echo ""
	./$app --operation service --option "kill-all"
	killproc $app
	rm -f /var/lock/subsys/$app
	echo
}

# Reload the service
reload() {
	status $app > /dev/null
	# If application is running
	if [[ $? -eq 0 ]]; then
		echo -n $"Reloading $app:"
		kill -HUP %s
		sleep 1
		status $app > /dev/null
		# If application is running
		if [[ $? -eq 0 ]]; then
			success
			./$app --operation service --option "status"
			echo
		else
			failure
			echo
		fi
	else
		run
	fi
}

help() {
	./$app --operation service --option "help"
}

applicationStatus() {
	./$app --operation service --option "status"
}

statusOf() {
	./$app --operation service --option "status-of $1"
}

pause() {
	./$app --operation service --option "pause $1"
}

pauseGroup() {
	./$app --operation service --option "pause-group $1"
}

pauseAll() {
	./$app --operation service --option "pause-all"
}

unpause() {
	./$app --operation service --option "unpause $1"
}

unpauseGroup() {
	./$app --operation service --option "unpause-group $1"
}

unpauseAll() {
	./$app --operation service --option "unpause-all"
}

killAll() {
	stop
}

# Main logic
case "$1" in
	start)
		start
		applicationStatus
		;;
	stop)
		stop
		;;
	status)
		status $app
		applicationStatus
		;;
	restart)
		stop
		sleep 1
		start
		;;
	reload)
		reload
		;;
	status-of)
		statusOf $2
		;;
	pause)
		pause $2
		;;
	pause-group)
		pauseGroup $2
		;;
	pause-all)
		pauseAll
		;;
	unpause)
		unpause $2
		;;
	unpause-group)
		unpauseGroup $2
		;;
	unpause-all)
		unpauseAll
		;;
	kill-all)
		killAll
		;;
	*)
		echo $"Usage: $0 {start | stop | restart | reload | status | status-of <job name> | pause <job name> | pause-group <group name> | pause-all | unpause <job name> | unpause-group <group name> | unpause-all | kill-all}"
		exit 1
esac
exit 0
`
		serviceFile.WriteString(fmt.Sprintf(fileContent, chkconfig, description, serviceName, description, serviceName, currentDir, configFile, logFile, listeningPort, "`pidof $app`"))
		serviceFile.Sync()

		fmt.Println("Service installed.")

	} else {
		fmt.Println("Something is not right. Terminating")
		return
	}
}

func install() {
	var installMethodPicked string
	if *silentInstall == true {
		installMethodPicked = *installMethod
	} else {
		installMethodPicked = StringPrompt("How do you want to install? (servicectl | initd)")
	}

	switch installMethodPicked {
	case "servicectl":
		installAsServicectl()
	case "initd":
		installAsInitd()
	}
}

func uninstallServicectl() {
	//delete file under /etc/systemd/system/gormq-supervisor.service
	if _, err := os.Stat(pathFile); err == nil {
		err := os.Remove(pathFile)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println("Service removed. Please reload the services launching the following command:")
		fmt.Println("systemctl daemon-reload")
		return
	} else if errors.Is(err, os.ErrNotExist) {
		fmt.Printf("Service file at %v does not exists. Service might not be installed.\n", pathFile)
		return
	}
}

func uninstallInitd() {
	//delete file under /etc/systemd/system/gormq-supervisor.service
	if _, err := os.Stat(initDFile); err == nil {
		err := os.Remove(initDFile)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println("Service removed. Please reload the services launching the following command:")
		fmt.Println("systemctl daemon-reload")
		return
	} else if errors.Is(err, os.ErrNotExist) {
		fmt.Printf("Service file at %v does not exists. Service might not be installed.\n", initDFile)
		return
	}
}

func uninstall() {
	uninstallServicectl()
	uninstallInitd()
}
