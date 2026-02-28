package main

import (
	"context"
	"flag"
	"fmt"
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

var jobKiller JobKiller
var wg sync.WaitGroup
var log Logger

const VERSION = "v0.1"

var mainContext context.Context

func main() {
	mainContext = context.Background()
	flag.Parse()

	instruction := *operationInstruction
	if instruction != "" {
		usage := "Available operations: install | uninstall . Both operation need to be launched as sudo"
		switch instruction {
		case "install":
			install()
			os.Exit(0)
		case "uninstall":
			uninstall()
			os.Exit(0)
		case "service":
			commandLineService(*serviceCommand)
		default:
			fmt.Println(usage)
			os.Exit(0)
		}
	} else {
		log = Logger{Path: *logPath + "goncsupervisorlogs.txt"}
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

		configuration, err := createConfig(*configFile)
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

func worker(configuration ConfigFile) {
	mainPid := os.Getpid()
	for j := 0; j < len(configuration.Jobs); j++ {
		connectionConfig, err := configuration.getConnectionByName(configuration.Jobs[j].ConnectionName)
		if err != nil {
			log.Printf("Skipping job %q: connection %q not found in config\n",
				configuration.Jobs[j].Name, configuration.Jobs[j].ConnectionName)
			continue
		}

		wg.Add(1)
		configuration.Jobs[j].ConnectionConfig = *connectionConfig
		configuration.Jobs[j].MainPid = mainPid
		configuration.Jobs[j].OwnContext, configuration.Jobs[j].OwnContextCancel = context.WithCancel(mainContext)
		go configuration.Jobs[j].executeCommand(&wg)
		jobKiller.Jobs = append(jobKiller.Jobs, &configuration.Jobs[j])
	}
	go jobKiller.listening()

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
		return jobKiller.returnStatus()
	case "status-of":
		return jobKiller.returnStatusOf(arguments)
	case "pause":
		jobKiller.pause(arguments)
		return "Job will be paused after getting out of sleep cycle or after execution. Current status: \n" + jobKiller.returnStatusOf(arguments)
	case "pause-group":
		jobKiller.pauseGroup(arguments)
		return "Jobs will be paused after getting out of sleep cycle or after execution. Current status: \n" + jobKiller.returnStatus()
	case "pause-all":
		jobKiller.pauseAll()
		return "Jobs will be paused after getting out of sleep cycle or after execution. Current status: \n" + jobKiller.returnStatus()
	case "unpause":
		jobKiller.unpause(arguments)
		time.Sleep(1 * time.Second)
		return jobKiller.returnStatusOf(arguments)
	case "unpause-group":
		jobKiller.unpauseGroup(arguments)
		time.Sleep(1 * time.Second)
		return jobKiller.returnStatus()
	case "unpause-all":
		jobKiller.unpauseAll()
		time.Sleep(1 * time.Second)
		return jobKiller.returnStatus()
	case "kill-all":
		jobKiller.killAll()
		return jobKiller.returnStatus()
	case "version":
		return VERSION
	case "update-job":
		if len(inputCommand) < 4 {
			return "In order to update the job property you need to pass the job name, the property that you need to update and the new value, all separated by space."
		}
		jobName := inputCommand[1]
		job, err := jobKiller.findJobByName(jobName)
		if err != nil {
			return err.Error()
		}
		updateJobArguments := inputCommand[2:]
		err = job.updateProperties(updateJobArguments)
		if err != nil {
			return err.Error()
		}
		return "Job updated successfully. Current status: \n" + jobKiller.returnStatusOf(jobName)
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
