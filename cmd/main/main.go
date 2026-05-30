package main

import (
	"context"
	"flag"
	"fmt"
	"gormq-supervisor/internal/config"
	"gormq-supervisor/internal/installer"
	"gormq-supervisor/internal/job"
	"gormq-supervisor/internal/logger"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	configFile           = flag.String("config", "./gonc-config.json", "path of configuration file")
	operationInstruction = flag.String("operation", "", "Available operations: install | uninstall | service")
	serviceCommand       = flag.String("option", "", "Available options: status | status-of <job name> | pause <job name> | pause-group <group name> | pause-all | unpause <job name> | unpause-group <group name> | unpause-all | kill-all | version")
	logPath              = flag.String("log", "./", "path where to store logs")
	port                 = flag.String("port", "9000", "Port where the server should listen")
	testMode             = flag.Bool("testing", false, "")
	installMethod        = flag.String("installMethod", "servicectl", "Install method (servicectl | initd)")
	silentInstall        = flag.Bool("silent", false, "Install with default values")
)

var (
	stop             = make(chan struct{})
	done             = make(chan struct{})
	killAllProcesses = make(chan struct{})
)

var jobKiller job.JobKiller
var wg sync.WaitGroup
var log logger.Logger

const VERSION = "v0.3"

var mainContext context.Context

func main() {
	mainContext = context.Background()
	flag.Parse()

	instruction := *operationInstruction
	if instruction != "" {
		usage := "Available operations: install | uninstall . Both operation need to be launched as sudo"
		switch instruction {
		case "install":
			installer.Install(*silentInstall, *installMethod)
			os.Exit(0)
		case "uninstall":
			installer.Uninstall()
			os.Exit(0)
		case "service":
			commandLineService(*serviceCommand)
		default:
			fmt.Println(usage)
			os.Exit(0)
		}
	} else {
		log = logger.Logger{Path: *logPath + "goncsupervisorlogs.txt"}
		defer log.Close()

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			sig := <-sigs
			log.Printf("Received signal: %v\n", sig)
			stop <- struct{}{}
			log.Println("Terminating...")
			killAllProcesses <- struct{}{}
		}()
		log.Println("loading configuration")

		configuration, err := config.CreateConfig(*configFile)
		if err != nil {
			log.Printf("Failed to load configuration: %v\n", err)
			fmt.Printf("Failed to load configuration: %v\n", err)
			os.Exit(1)
		}
		log.Println("configuration loaded")

		log.Println("Starting workers")
		go worker(configuration)
		<-stop
		wg.Wait()
		log.Println("All jobs have stopped")
		log.Println("Terminated")
		log.Println("- - - - - - - - - - - - - - -")
		os.Exit(0)
	}
}

func listening() {
	for {
		time.Sleep(time.Second)
		select {
		case <-killAllProcesses:
			jobKiller.KillAll()
		}
	}
}

func worker(configuration config.ConfigFile) {
	mainPid := os.Getpid()
	for j := 0; j < len(configuration.Jobs); j++ {
		connectionConfig, err := configuration.GetConnectionByName(configuration.Jobs[j].ConnectionName)
		if err != nil {
			log.Printf("Skipping job %q: connection %q not found in config\n",
				configuration.Jobs[j].Name, configuration.Jobs[j].ConnectionName)
			continue
		}

		wg.Add(1)
		configuration.Jobs[j].ConnectionConfig = *connectionConfig
		configuration.Jobs[j].MainPid = mainPid
		configuration.Jobs[j].OwnContext, configuration.Jobs[j].OwnContextCancel = context.WithCancel(mainContext)
		configuration.Jobs[j].Log = log
		configuration.Jobs[j].TestMode = *testMode
		go configuration.Jobs[j].ExecuteCommand(&wg)
		jobKiller.Jobs = append(jobKiller.Jobs, &configuration.Jobs[j])
	}
	go listening()

	go server()

LOOP:
	for {
		time.Sleep(time.Second)
		select {
		case <-stop:
			break LOOP
		default:
		}
	}
	done <- struct{}{}
}

func server() {
	// Listen for incoming connections.
	l, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		log.Printf("Error listening to port %s: %v\n", *port, err)
		fmt.Printf("Error listening to port %s: %v\n", *port, err)
		return
	}
	// Close the listener when the application closes.
	defer l.Close()

	log.Printf("Server listening on port %s\n", *port)

	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			// Check if the error is due to listener being closed
			if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
				return
			}
			log.Printf("Error accepting connection: %v\n", err)
			continue
		}
		// Handle connections in a new goroutine.
		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	for {
		buf := make([]byte, 1024)
		size, err := conn.Read(buf)
		if err != nil {
			return
		}
		data := buf[:size]
		response := createResponse(string(data))
		conn.Write([]byte(response))
		conn.Close()
	}
}

func createResponse(command string) string {
	inputCommand := strings.Fields(command)
	if len(inputCommand) == 0 {
		return "Commands available:\nstatus | status-of <job name> | pause <job name> | pause-group <group name> | pause-all | unpause <job name> | unpause-group <group name> | unpause-all | kill-all | version\n"
	}
	action := inputCommand[0]
	arguments := ""
	if len(inputCommand) > 1 {
		arguments = strings.Join(inputCommand[1:], " ")
	}
	switch action {
	case "status":
		return jobKiller.ReturnStatus()
	case "status-of":
		return jobKiller.ReturnStatusOf(arguments)
	case "pause":
		jobKiller.Pause(arguments)
		return "Job will be paused after getting out of sleep cycle or after execution. Current status: \n" + jobKiller.ReturnStatusOf(arguments)
	case "pause-group":
		jobKiller.PauseGroup(arguments)
		return "Jobs will be paused after getting out of sleep cycle or after execution. Current status: \n" + jobKiller.ReturnStatus()
	case "pause-all":
		jobKiller.PauseAll()
		return "Jobs will be paused after getting out of sleep cycle or after execution. Current status: \n" + jobKiller.ReturnStatus()
	case "unpause":
		jobKiller.Unpause(arguments)
		time.Sleep(1 * time.Second)
		return jobKiller.ReturnStatusOf(arguments)
	case "unpause-group":
		jobKiller.UnpauseGroup(arguments)
		time.Sleep(1 * time.Second)
		return jobKiller.ReturnStatus()
	case "unpause-all":
		jobKiller.UnpauseAll()
		time.Sleep(1 * time.Second)
		return jobKiller.ReturnStatus()
	case "kill-all":
		jobKiller.KillAll()
		return jobKiller.ReturnStatus()
	case "version":
		return VERSION
	case "update-job":
		if len(inputCommand) < 4 {
			return "In order to update the job property you need to pass the job name, the property that you need to update and the new value, all separated by space."
		}
		jobName := inputCommand[1]
		job, err := jobKiller.FindJobByName(jobName)
		if err != nil {
			return err.Error()
		}
		updateJobArguments := inputCommand[2:]
		err = job.UpdateProperties(updateJobArguments)
		if err != nil {
			return err.Error()
		}
		return "Job updated successfully. Current status: \n" + jobKiller.ReturnStatusOf(jobName)
	default:
		return "Commands available:\nstatus | status-of <job name> | pause <job name> | pause-group <group name> | pause-all | unpause <job name> | unpause-group <group name> | unpause-all | kill-all | version\n"
	}
}

func commandLineService(command string) {
	endpoint := "localhost:" + *port
	connection, err := net.Dial("tcp", endpoint)
	if err != nil {
		fmt.Printf("Failed to connect to service at %s: %v\n", endpoint, err)
		return
	}
	defer connection.Close()

	_, err = connection.Write([]byte(command))
	if err != nil {
		fmt.Printf("Failed to send command: %v\n", err)
		return
	}

	buffer := make([]byte, 4096)
	n, err := connection.Read(buffer)
	if err != nil {
		fmt.Printf("Failed to read response: %v\n", err)
		return
	}

	fmt.Print(string(buffer[:n]))
}
