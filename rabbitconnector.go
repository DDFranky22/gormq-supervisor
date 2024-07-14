package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	Endpoint string
	Username string
	Password string
}

type QueueInfo struct {
	Messages int `json:"messages"`
}

func createClient(Endpoint string, Username string, Password string) *Client {

	client := Client{
		Endpoint: Endpoint,
		Username: Username,
		Password: Password,
	}

	return &client
}

func (client *Client) getQueue(Vhost string, QueueName string) (*QueueInfo, error) {
	httpClient := &http.Client{
		Timeout: time.Second * 10,
	}

	apiEndpoint := client.Endpoint + "/api/queues/" + url.QueryEscape(Vhost) + "/" + url.QueryEscape(QueueName)

	req, err := http.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(client.Username, client.Password)
	response, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("can't recover information for queue %v on virtual host %v", QueueName, Vhost)
	}

	var queueInfo QueueInfo
	json.NewDecoder(response.Body).Decode(&queueInfo)

	return &queueInfo, nil
}

func (client *Client) getMessages(job *Job) (int, bool) {
	if *testing {
		return 1, true
	}
	q, err := client.getQueue(job.ConnectionConfig.Vhost, job.Queue)
	if err != nil {
		var arrayOutput []string
		output := fmt.Sprintf("Can't connect to queue: %v on vhost: %v - Error: %v", job.Queue, job.ConnectionConfig.Vhost, err)
		arrayOutput = append(arrayOutput, output)
		job.logOutput(arrayOutput)
		return 0, false
	}

	return q.Messages, true
}
