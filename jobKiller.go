package main

import (
	"bytes"
	"fmt"
	"text/tabwriter"
	"time"
)

type JobKiller struct {
	Jobs []*Job
}

func (jobKiller JobKiller) listening() {
	for {
		time.Sleep(time.Second)
		select {
		case <-killAllProcesses:
			//jobKiller.stopAll()
			jobKiller.killAll()
		}
	}
}

func (jobKiller *JobKiller) pauseAll() {
	for i := 0; i < len(jobKiller.Jobs); i++ {
		if !jobKiller.Jobs[i].Pause {
			jobKiller.Jobs[i].Pause = true
		}
	}
}

func (jobKiller *JobKiller) pause(jobName string) {
	for i := 0; i < len(jobKiller.Jobs); i++ {
		if !jobKiller.Jobs[i].Pause && jobKiller.Jobs[i].Name == jobName {
			jobKiller.Jobs[i].Pause = true
			break
		}
	}
}

func (jobKiller *JobKiller) pauseGroup(groupName string) {
	for i := 0; i < len(jobKiller.Jobs); i++ {
		for _, b := range jobKiller.Jobs[i].Groups {
			if b == groupName {
				jobKiller.Jobs[i].Pause = true
				break
			}
		}
	}
}

func (jobKiller *JobKiller) unpauseAll() {
	for i := 0; i < len(jobKiller.Jobs); i++ {
		if jobKiller.Jobs[i].Pause {
			jobKiller.Jobs[i].Pause = false
		}
	}
}

func (jobKiller *JobKiller) unpause(jobName string) {
	for i := 0; i < len(jobKiller.Jobs); i++ {
		if jobKiller.Jobs[i].Pause && jobKiller.Jobs[i].Name == jobName {
			jobKiller.Jobs[i].Pause = false
			break
		}
	}
}

func (jobKiller *JobKiller) unpauseGroup(groupName string) {
	for i := 0; i < len(jobKiller.Jobs); i++ {
		for _, b := range jobKiller.Jobs[i].Groups {
			if b == groupName {
				jobKiller.Jobs[i].Pause = false
				break
			}
		}
	}
}

func (jobKiller *JobKiller) killAll() {
	for i := 0; i < len(jobKiller.Jobs); i++ {
		jobKiller.Jobs[i].Stop = true
		jobKiller.Jobs[i].OwnContextCancel()
		if jobKiller.Jobs[i].PID != 0 {
			err := jobKiller.Jobs[i].CmdExecutable.Process.Kill()
			if err != nil {
				fmt.Println(err)
			}
		}
		jobKiller.Jobs[i].Status = STATUS_TERMINATED
	}
}

func (jobKiller *JobKiller) returnStatus() string {
	var b bytes.Buffer
	writer := tabwriter.NewWriter(&b, 10, 0, 2, ' ', tabwriter.Debug)
	fmt.Fprintf(writer, "%v\t%v\t%v\t%v\t%v\t%v\t%v\n", "Job", "Groups", "Status", "PID", "User", "Sleep", "Last Exec")
	for i := 0; i < len(jobKiller.Jobs); i++ {
		job := jobKiller.Jobs[i]
		jobStatus := job.getStatus()
		fmt.Fprintf(writer, "%v\t%v\t%v\t%v\t%v\t%v\t%v\n", jobStatus["Name"], jobStatus["Groups"], jobStatus["Status"], jobStatus["PID"], jobStatus["User"], jobStatus["Sleep"], jobStatus["LastExec"])
	}
	writer.Flush()
	return b.String()
}

func (jobKiller *JobKiller) returnStatusOf(jobName string) string {
	var b bytes.Buffer
	writer := tabwriter.NewWriter(&b, 10, 0, 2, ' ', tabwriter.Debug)
	fmt.Fprintf(writer, "%v\t%v\t%v\t%v\t%v\t%v\t%v\n", "Job", "Groups", "Status", "PID", "User", "Sleep", "Last Exec")
	found := false
	for i := 0; i < len(jobKiller.Jobs); i++ {
		job := jobKiller.Jobs[i]
		if job.Name == jobName {
			found = true
			jobStatus := job.getStatus()
			fmt.Fprintf(writer, "%v\t%v\t%v\t%v\t%v\t%v\t%v\n", jobStatus["Name"], jobStatus["Groups"], jobStatus["Status"], jobStatus["PID"], jobStatus["User"], jobStatus["Sleep"], jobStatus["LastExec"])
			break
		}
	}

	if found {
		writer.Flush()
		return b.String()
	}
	return fmt.Sprintf("Can't find job called %v\n", jobName)
}
