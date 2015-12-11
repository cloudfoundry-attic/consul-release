package turbulence

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type Client struct {
	baseURL          string
	operationTimeout time.Duration
}

type deployment struct {
	Name string
	Jobs []job
}

type job struct {
	Name    string
	Indices []int
}

type killTask struct {
	Type string
}

type killCommand struct {
	Tasks       []interface{}
	Deployments []deployment
}

type Response struct {
	ID                   string           `json:"ID"`
	ExecutionCompletedAt string           `json:"ExecutionCompletedAt"`
	Events               []*ResponseEvent `json:"Events"`
}

type ResponseEvent struct {
	Error string `json:"Error"`
}

func NewClient(baseURL string) Client {
	return Client{
		baseURL:          baseURL,
		operationTimeout: 5 * time.Minute,
	}
}

func (c Client) KillIndices(deploymentName, jobName string, indices []int) error {
	command := killCommand{
		Tasks: []interface{}{
			killTask{Type: "kill"},
		},
		Deployments: []deployment{{
			Name: deploymentName,
			Jobs: []job{{Name: jobName, Indices: indices}},
		}},
	}

	jsonCommand, err := json.Marshal(command)
	if err != nil {
		return err
	}

	request, err := http.NewRequest("POST", c.baseURL+"/api/v1/incidents", bytes.NewBuffer(jsonCommand))
	if err != nil {
		return err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Do(request)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	turbulenceResponse := new(Response)
	err = json.Unmarshal(body, turbulenceResponse)
	if err != nil {
		return err
	}

	return c.pollRequestCompletedDeletingVM(turbulenceResponse.ID)
}

func (c Client) pollRequestCompletedDeletingVM(id string) error {
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/incidents/%s", c.baseURL, id), nil)
	if err != nil {
		return err
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	startTime := time.Now()
	for {
		resp, err := client.Do(request)
		if err != nil {
			return err
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		turbulenceResponse := new(Response)
		err = json.Unmarshal(body, turbulenceResponse)
		if err != nil {
			return err
		}

		if turbulenceResponse.ExecutionCompletedAt != "" {
			if len(turbulenceResponse.Events) == 0 {
				return errors.New("There should at least be one Event in response from turbulence.")
			}

			for _, event := range turbulenceResponse.Events {
				if event.Error != "" {
					return errors.New(event.Error)
				}
			}

			return nil
		}

		if time.Now().Sub(startTime) > c.operationTimeout {
			return errors.New(fmt.Sprintf("Did not finish deleting VM in time: %d", c.operationTimeout))
		}

		time.Sleep(2 * time.Second)
	}

	return nil
}
