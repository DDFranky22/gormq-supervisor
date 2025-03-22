package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Job struct {
	Name              string   `json:"name"`
	Groups            []string `json:"groups"`
	SleepTime         int      `json:"sleep_time"`
	SleepIncrement    int      `json:"sleep_increment"`
	MaxSleep          int      `json:"max_sleep"`
	MinMessages       int      `json:"min_messages"`
	WorkingDir        string   `json:"working_dir"`
	UserId            string   `json:"user"`
	Command           string   `json:"command"`
	Spawn             int      `json:"spawn"`
	ConnectionName    string   `json:"connection"`
	Queue             string   `json:"queue"`
	ErrorLogPath      string   `json:"error_log_path"`
	ErrorLogMaxKBSize float64  `json:"error_log_max_kb_size"`
	ErrorLogMaxFiles  int      `json:"error_log_max_files"`
	MaxExecution      int64    `json:"max_execution"`
	PID               int
	MainPid           int
	CurrentSleepTime  int
	ConnectionConfig  ConnectionConfig
	CmdExecutable     *exec.Cmd
	Status            int16
	Stop              bool
	Pause             bool
	StartedAt         int64
	OwnContext        context.Context
	OwnContextCancel  context.CancelFunc
}

const STATUS_SLEEP = 0
const STATUS_RUNNING = 1
const STATUS_PAUSED = 2
const STATUS_TERMINATED = 3

func (job *Job) getStatus() map[string]interface{} {
	statusContainer := make(map[string]interface{})
	statusContainer["Name"] = job.Name
	statusContainer["Groups"] = job.Groups
	statusContainer["Status"] = job.getStatusName()
	statusContainer["PID"] = job.PID
	statusContainer["User"] = job.UserId
	statusContainer["Sleep"] = job.CurrentSleepTime
	statusContainer["MaxSleep"] = job.MaxSleep
	statusContainer["LastExec"] = time.Unix(job.StartedAt, 0)
	return statusContainer
}

func (job *Job) getStatusName() string {
	switch job.Status {
	case STATUS_SLEEP:
		return "SLEEPING"
	case STATUS_RUNNING:
		return "RUNNING"
	case STATUS_PAUSED:
		return "PAUSED"
	case STATUS_TERMINATED:
		return "TERMINATED"
	default:
		return "UNKNOWN"
	}
}

func (job *Job) executeCommand(wg *sync.WaitGroup) {
	defer wg.Done()
	log.Println("Starting Job: " + job.Name)
	rmqc := createClient(job.ConnectionConfig.Endpoint, job.ConnectionConfig.Username, job.ConnectionConfig.Password)
	runningUserId, err := job.returnUserId()
	if err != nil {
		log.Printf("For job: \"%v\" could not recover user \"%v\". Cannot be executed\n", job.Name, job.UserId)
	}
	runningUserMainGroup, runningUserGroups, err := job.returnUserGroups()
	if err != nil {
		log.Printf("For job: \"%v\" could not recover groups for user \"%v\". Cannot be executed\n", job.Name, job.UserId)
	}
LOOP:
	for {
		if job.Stop {
			break LOOP
		}
		if !job.checkIfStillActive(job.MainPid) {
			break LOOP
		}
		if job.Pause {
			if job.Status != STATUS_PAUSED {
				job.Status = STATUS_PAUSED
				job.CurrentSleepTime = job.SleepTime
			}
			time.Sleep(1 * time.Second)
			continue
		}
		job.Status = STATUS_SLEEP
		queueMessages, execute := rmqc.getMessages(job)
		if execute {
			if job.MinMessages <= queueMessages {
				job.Status = STATUS_RUNNING
				stringCommand := strings.Fields(job.Command)
				app, minusApp := stringCommand[0], stringCommand[1:]
				commandContext := context.Background()
				if job.MaxExecution > 0 {
					var cancelCommandContext context.CancelFunc
					commandContext, cancelCommandContext = context.WithTimeout(context.Background(), time.Duration(job.MaxExecution)*time.Second)
					defer cancelCommandContext()
				}
				cmd := exec.CommandContext(commandContext, app, minusApp...)
				if job.WorkingDir != "" {
					absolutePath, error := filepath.Abs(job.WorkingDir)
					if error != nil {
						log.Printf("For job: \"%v\" the directory \"%v\" does not exists. Cannot be executed\n", job.Name, job.WorkingDir)
						break LOOP
					}
					cmd.Dir = absolutePath
				}
				if runningUserId != 0 && runningUserMainGroup != 0 {
					cmd.SysProcAttr = &syscall.SysProcAttr{}
					cmd.SysProcAttr.Credential = &syscall.Credential{
						Uid:         runningUserId,
						Gid:         runningUserMainGroup,
						Groups:      runningUserGroups,
						NoSetGroups: false,
					}
				}
				job.CmdExecutable = cmd
				stdout, _ := cmd.StdoutPipe()
				stderr, _ := cmd.StderrPipe()
				startErr := cmd.Start()
				if startErr != nil {
					log.Printf("For job: \"%v\" the command: \"%v\" cannot be executed. Output: %v\n", job.Name, job.Command, startErr)
					break LOOP
				}
				now := time.Now()
				job.StartedAt = now.Unix()
				job.PID = cmd.Process.Pid
				if job.ErrorLogPath != "" {
					scanner := bufio.NewScanner(io.MultiReader(stdout, stderr))
					var output []string
					scanner.Split(bufio.ScanWords)
					for scanner.Scan() {
						output = append(output, scanner.Text())
					}
					job.logOutput(output)
				}
				cmd.Wait()
				if commandContext.Err() == context.DeadlineExceeded {
					var deadlineOutput []string
					job.logOutput(append(deadlineOutput, fmt.Sprintf("Job \"%v\" exceeded max execution time of %v seconds. Process Killed.", job.Name, job.MaxExecution)))
				}
				job.PID = 0
				job.CurrentSleepTime = job.SleepTime
			}
		}
		job.Sleep(job.OwnContext)
		newSleepTime := job.CurrentSleepTime + job.SleepIncrement
		if newSleepTime >= job.MaxSleep {
			newSleepTime = job.MaxSleep
		}
		job.CurrentSleepTime = newSleepTime
	}
	job.Status = STATUS_TERMINATED
	log.Println("Ending Job: " + job.Name)
}

func (job *Job) logFolder() (string, error) {
	if job.ErrorLogPath != "" {
		logFolder := job.ErrorLogPath + job.Name
		if _, err := os.Stat(logFolder); os.IsNotExist(err) {
			err := os.Mkdir(logFolder, 0760)
			if err != nil {
				log.Println("Error in making log folder")
				log.Println(err)
				return "", err
			}
		}
		return logFolder, nil
	}
	return "", nil
}

func (job *Job) getLogFile(logFolder string) (*os.File, error) {
	now := time.Now()
	dirEntries, err := os.ReadDir(logFolder)
	if err != nil {
		log.Println("Error in reading log folder")
		log.Println(err)
		return nil, err
	}
	files := []string{}
	for _, entry := range dirEntries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}
	var logFile *os.File
	if len(files) > 0 {
		filesArray := files
		sort.Slice(filesArray, func(i, j int) bool {
			return filesArray[i] < filesArray[j]
		})
		loggingFileName := filesArray[len(filesArray)-1]
		if job.ErrorLogMaxFiles >= 1 && job.ErrorLogMaxFiles < len(filesArray) {
			err := os.Remove(logFolder + "/" + filesArray[0])
			if err != nil {
				log.Println("Can't remove file")
				log.Println(err)
				return nil, err
			}
		}

		logFile, err = os.OpenFile(logFolder+"/"+loggingFileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil || logFile == nil {
			log.Printf("Can't open log file %v\n", logFolder+"/"+loggingFileName)
		}

		if job.ErrorLogMaxKBSize > 0 {
			logFileStats, err := os.Stat(logFile.Name())
			if err != nil {
				log.Println("Can't get stats of log file")
				return nil, err
			}

			if float64(logFileStats.Size()) >= (job.ErrorLogMaxKBSize * 1024) {
				logName := strconv.FormatInt(now.Unix(), 10) + "_log.txt"
				newLogPath := logFolder + "/" + logName
				logFile, err = os.Create(newLogPath)
				if err != nil {
					log.Println(err)
					return nil, err
				}
			}
		}

	} else {
		logName := strconv.FormatInt(now.Unix(), 10) + "_log.txt"
		newLogPath := logFolder + "/" + logName
		logFile, err = os.Create(newLogPath)
		if err != nil {
			log.Println(err)
			return nil, err
		}
	}

	return logFile, nil
}

func (job *Job) logOutput(output []string) {
	if job.ErrorLogPath != "" {
		now := time.Now()
		formatted := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
			now.Year(), now.Month(), now.Day(),
			now.Hour(), now.Minute(), now.Second())
		jointOutput := strings.Join(output, " ")
		if jointOutput == "" {
			return
		}
		logString := "[" + formatted + "] " + jointOutput + "\n"
		logFolder, err := job.logFolder()
		if err != nil {
			return
		}

		var logFile *os.File
		logFile, err = job.getLogFile(logFolder)
		if err != nil {
			log.Println("Error in getting log file")
			log.Println(err)
			return
		}

		_, err = logFile.WriteString(logString)
		if err != nil {
			log.Println("Can't log")
			log.Println(err)
			return
		}
		logFile.Sync()
		defer logFile.Close()
	}
}

func (job *Job) checkIfStillActive(pid int) bool {
	_, err := os.FindProcess(int(pid))
	if err != nil {
		log.Println("Error")
		log.Println(err)
		return false
	}
	return true
}

func (job *Job) Sleep(ctx context.Context) error {
	var timer = time.NewTimer(time.Duration(job.CurrentSleepTime) * time.Second)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (job *Job) clone(numberItem int) Job {
	newJob := Job{
		job.Name + "_" + strconv.Itoa(numberItem),
		job.Groups,
		job.SleepTime,
		job.SleepIncrement,
		job.MaxSleep,
		job.MinMessages,
		job.WorkingDir,
		job.UserId,
		job.Command,
		1,
		job.ConnectionName,
		job.Queue,
		job.ErrorLogPath,
		job.ErrorLogMaxKBSize,
		job.ErrorLogMaxFiles,
		job.MaxExecution,
		0,
		0,
		0,
		ConnectionConfig{},
		nil,
		0,
		false,
		false,
		0,
		nil,
		nil,
	}

	return newJob
}

func (job *Job) returnUserId() (uint32, error) {
	var runningUserId uint64 = 0
	if job.UserId != "" {
		findUserCommand := exec.Command("id", "-u", job.UserId)
		output, err := findUserCommand.Output()
		if err != nil {
			return 0, err
		}
		runningUserId, err = strconv.ParseUint(strings.TrimSpace(string(output)), 10, 32)
		if err != nil {
			return 0, err
		}
	}
	return uint32(runningUserId), nil
}

func (job *Job) returnUserGroups() (uint32, []uint32, error) {
	var groups = []uint32{}
	if job.UserId != "" {
		findUserGroups := exec.Command("id", "-G", job.UserId)
		output, err := findUserGroups.Output()
		if err != nil {
			return 0, []uint32{}, err
		}
		returnedGroups := strings.Fields(strings.TrimSpace(string(output)))
		if len(returnedGroups) <= 0 {
			return 0, []uint32{}, err
		}
		for _, value := range returnedGroups {
			convertedValue, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return 0, []uint32{}, err
			}
			groups = append(groups, uint32(convertedValue))
		}
	}
	if len(groups) > 0 {
		return groups[0], groups, nil
	}
	return 0, groups, nil
}

func (job *Job) updateProperties(properties []string) error {
	propertyToUpdate := properties[0]
	switch propertyToUpdate {
	case "min_messages":
		newMinMessages, err := strconv.Atoi(properties[1])
		if err != nil {
			return err
		}
		if newMinMessages < 0 {
			return errors.New("You cannot set a negative value")
		}
		job.MinMessages = newMinMessages
	case "sleep_time":
		newSleepTime, err := strconv.Atoi(properties[1])
		if err != nil {
			return err
		}
		if newSleepTime < 0 {
			return errors.New("You cannot set a negative value")
		}
		job.SleepTime = newSleepTime
	case "sleep_increment":
		newSleepIncrement, err := strconv.Atoi(properties[1])
		if err != nil {
			return err
		}
		if newSleepIncrement < 0 {
			return errors.New("You cannot set a negative value")
		}
		job.SleepIncrement = newSleepIncrement
	case "max_sleep":
		newMaxSleep, err := strconv.Atoi(properties[1])
		if err != nil {
			return err
		}
		if newMaxSleep < 0 {
			return errors.New("You cannot set a negative value")
		}
		job.MaxSleep = newMaxSleep
	case "spawn":
		newSpawn, err := strconv.Atoi(properties[1])
		if err != nil {
			return err
		}
		if newSpawn <= 0 {
			return errors.New("You cannot set a negative value or 0")
		}
		job.Spawn = newSpawn
	case "max_execution":
		newMaxExecution, err := strconv.Atoi(properties[1])
		if err != nil {
			return err
		}
		if newMaxExecution < 0 {
			return errors.New("You cannot set a negative value")
		}
		job.MaxExecution = int64(newMaxExecution)
	default:
		return errors.New("Property not supported. The supported properties are: min_messages | sleep_time | sleep_increment | max_sleep | max_execution | spawn")
	}
	return nil
}
