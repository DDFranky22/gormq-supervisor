package job

import (
	"bytes"
	"errors"
	"fmt"
	"text/tabwriter"
)

type JobKiller struct {
	Jobs []*Job
}

func (jobKiller *JobKiller) PauseAll() {
	for i := 0; i < len(jobKiller.Jobs); i++ {
		if !jobKiller.Jobs[i].GetPause() {
			jobKiller.Jobs[i].SetPause(true)
		}
	}
}

func (jobKiller *JobKiller) Pause(jobName string) {
	for i := 0; i < len(jobKiller.Jobs); i++ {
		if !jobKiller.Jobs[i].GetPause() && jobKiller.Jobs[i].Name == jobName {
			jobKiller.Jobs[i].SetPause(true)
			break
		}
	}
}

func (jobKiller *JobKiller) PauseGroup(groupName string) {
	for i := 0; i < len(jobKiller.Jobs); i++ {
		for _, b := range jobKiller.Jobs[i].Groups {
			if b == groupName {
				jobKiller.Jobs[i].SetPause(true)
				break
			}
		}
	}
}

func (jobKiller *JobKiller) UnpauseAll() {
	for i := 0; i < len(jobKiller.Jobs); i++ {
		if jobKiller.Jobs[i].GetPause() {
			jobKiller.Jobs[i].SetPause(false)
		}
	}
}

func (jobKiller *JobKiller) Unpause(jobName string) {
	for i := 0; i < len(jobKiller.Jobs); i++ {
		if jobKiller.Jobs[i].GetPause() && jobKiller.Jobs[i].Name == jobName {
			jobKiller.Jobs[i].SetPause(false)
			break
		}
	}
}

func (jobKiller *JobKiller) UnpauseGroup(groupName string) {
	for i := 0; i < len(jobKiller.Jobs); i++ {
		for _, b := range jobKiller.Jobs[i].Groups {
			if b == groupName {
				jobKiller.Jobs[i].SetPause(false)
				break
			}
		}
	}
}

func (jobKiller *JobKiller) KillAll() {
	for i := 0; i < len(jobKiller.Jobs); i++ {
		jobKiller.Jobs[i].SetStop(true)
		jobKiller.Jobs[i].OwnContextCancel()
		if jobKiller.Jobs[i].GetPID() != 0 {
			cmd := jobKiller.Jobs[i].GetCmdExecutable()
			if cmd != nil && cmd.Process != nil {
				err := cmd.Process.Kill()
				if err != nil {
					fmt.Println(err)
				}
			}
		}
		jobKiller.Jobs[i].SetStatus(STATUS_TERMINATED)
	}
}

func (jobKiller *JobKiller) ReturnStatus() string {
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

func (jobKiller *JobKiller) ReturnStatusOf(jobName string) string {
	var b bytes.Buffer
	writer := tabwriter.NewWriter(&b, 10, 0, 2, ' ', tabwriter.Debug)
	fmt.Fprintf(writer, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n", "Job", "Groups", "Status", "PID", "User", "Sleep", "Max sleep", "Last Exec")
	found := false
	for i := 0; i < len(jobKiller.Jobs); i++ {
		job := jobKiller.Jobs[i]
		if job.Name == jobName {
			found = true
			jobStatus := job.getStatus()
			fmt.Fprintf(writer, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n", jobStatus["Name"], jobStatus["Groups"], jobStatus["Status"], jobStatus["PID"], jobStatus["User"], jobStatus["Sleep"], jobStatus["MaxSleep"], jobStatus["LastExec"])
			break
		}
	}

	if found {
		writer.Flush()
		return b.String()
	}
	return fmt.Sprintf("Can't find job called %v\n", jobName)
}

func (jobKiller *JobKiller) FindJobByName(jobName string) (*Job, error) {
	found := false
	var jobToReturn *Job
	for i := 0; i < len(jobKiller.Jobs); i++ {
		job := jobKiller.Jobs[i]
		if job.Name == jobName {
			found = true
			jobToReturn = job
			break
		}
	}

	if found {
		return jobToReturn, nil
	}
	return nil, errors.New("Cannot find job with provided name")
}
