# gormq-supervisor
A program in go that launches commands (jobs) based on configuration only if there are messages in `RabbitMQ`

## why does this exists?
I found myself in a situation where I had multiple queues on RabbitMQ (20+) and the impossibility to have a program that constantly listen on the queue for incoming messages.
The first approach was to use crontab to execute a bash file with the processes that needs to be executed every 1 minute.
The second solution was to use a program like supervisor to constantly launch processes that interrogate RabbitMQ and then execute if there were any messages in the queue. 
However with the increase of the number of queues, for both solutions, it also meant an increased CPU load, and the processes will start even if there were no messages in queue.
supervisor also has the drawback that if there are no messages in queue and the process exists (due to natural timeout), it will not wait up to a minute like with crontab but it will spin the process back up immediatly. This is by design (and correct) for supervisor, and is kind of a pain in my situation.

Therefore I needed to create a program that:
- retrieved the number of messages in a specific queue
- if there is a message (or the designated number of messages) will start the process

All of this could have been avoided writing a program that will constantly listen to incoming messages for a specific exchange, but I was out of luck. Maybe this is also your case.
Hope this helps.

## dependencies
This program depends on:
- RabbitMQ > 3.6.0
- RabbitMQ management plugin
- port `9000` (can be configured)

It uses the management plugin HTTP API to retrieve the number of messages in a specific queue

## terminology
There are two main resources in this program:
- Connections
- Jobs
### Connections
Simply put, the connection to the RabbitMQ instance. They are composed as such:
- Name: custom name of the connection. This should be unique since it will be used to indicate the connection to a singol job
- Endpoint: URL for the RabbitMQ management plugin (usually same endpoint of RabbitMQ but with port 15672)
- Username: username to use when calling the API. It can be an environment variable in the form `${VARIABLE_NAME}`
- Password: password to use when calling the API. It can be an environment variable in the form `${VARIABLE_NAME}`
- Vhost: virtual host to use when calling the API.
### Jobs
These are the actual command that are run when there are the specified number of messages.
Here is their composition (those with an * are required in configuration):
- Name *: identifying name for the job
- Groups: list of groups to associate to the job
- SleepTime *: how much the program should wait to check on RabbitMQ for incoming messages
- SleepIncrement *: how much should the program increase the SleepTime every time there are no messages
- MaxSleep *: what is the maximum amount of sleep time for the program to check on incoming messages
- MinMessages *: minimum number of messages in order to execute the Command
- WorkingDir: specify the working directory for the command (if used in conjunction with the option User, be sure that user has the right permissions to navigate in the specified directory)
- User: specify a username (linux user) and use it to launch the command. In order for this to work, you need to launch this program as `root`
- Command *: command to launch when the conditions are met (single command for now, no concatenation)
- Spawn: number of jobs to spawn in order to have multiple consumers
- Connection *: Name of the connection to use
- Queue *: name of the queue to interrogate
- ErrorLogPath: path where to store errors/output of the command launched. This is a path to a folder, there the program will create a subfolder of its own.
- ErrorLogMaxKBSize: max size in KB for the error/output file
- ErrorLogMaxFiles: max number of files allowed for error/output. When reached, it will delete the oldest file.
- MaxExecution: max execution time allowed for the command. When reached, it will try to kill the command and reset the execution of the job

## sample configuration
```JSON
{
  "connections": [
    {
      "name": "default",
      "endpoint": "http://localhost:15672",
      "username": "${RABBIT_USER}",
      "password": "${RABBIT_PASSWORD}",
      "vhost": "/"
    }
  ],
  "jobs": [
    {
      "name": "job1",
      "groups": ["group1"],
      "sleep_time": 1,
      "sleep_increment": 1,
      "max_sleep": 10,
      "min_messages": 1,
      "working_dir": "./",
      "user": "apache",
      "command": "whoami",
      "spawn": 1,
      "connection": "default",
      "queue": "job1_test",
      "error_log_path": "./",
      "error_log_max_kb_size": 500,
      "error_log_max_files": 5,
      "max_execution": 10
    }
  ]
}
```

## how to run it
After downloading/cloning the repo, you can simply run:

```shell
go run *.go --config ./path_to_json_config --log ./
```

There are the flags available:
- `config`: path to the config file
- `log`: path to the generic log of the program
- `port`: specify the port where the service should listn (default `9000`)
- `testing`: used for testing and avoid calling RabbitMQ
- `operation`: this program comes with a fable attempt to "install" it as a service, either as `servicectl` or `initd`. It just means it creates one of two files base on the `installMethod` option.
- `installMethod`: attempt to install the program as a service. Needs to be `root`. The installation will be "interactive" by default
- `silent`: attempt to install with default values and will not ask anything when installing
- `option`: when used in conjunction with `operation` with value `service`, allows you to communicate with the main instance of the service via the specified port. This is used to show "status" of the jobs, pausing them and stopping them.

I suggest checking the `option` flag and its uses. With that you can pause, restart, check the status and even kill all the jobs.
This is possible both for singular jobs than for a group of jobs.
Here some examples:
```shell
go run *.go --operation service --option status-of job1
```
```shell
go run *.go --operation service --option pause job1
```
```shell
go run *.go --operation service --option unpause job1
```
```shell
go run *.go --operation service --option pause-group group1
```
```shell
go run *.go --operation service --option unpause-all
```

If you install this as the `initd` method you don't need to run the program itself, but you can simply run
```shell
service gormq-supervisor status
```
this will be the same with all the options available. If you used `systemctl`, sorry no can do.

## how to build it
If you want to build this program, this is the minimum command:
```shell
CGO_ENABLED=0 go build *.go
```
I suggest specifying also the distribution and architecture.
