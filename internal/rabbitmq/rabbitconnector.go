package rabbitmq

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
	TestMode bool
}

type QueueInfo struct {
	Messages int `json:"messages"`
}

func CreateClient(Endpoint string, Username string, Password string, TestMode bool) *Client {

	client := Client{
		Endpoint: Endpoint,
		Username: Username,
		Password: Password,
		TestMode: TestMode,
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

func (client *Client) GetMessages(vhost string, queue string) (int, bool) {
	if client.TestMode {
		return 1, true
	}
	q, err := client.getQueue(vhost, queue)
	if err != nil {
		return 0, false
	}

	return q.Messages, true
}
