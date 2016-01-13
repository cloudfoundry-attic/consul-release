package bosh

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/cloudfoundry-incubator/candiedyaml"
)

var (
	client     = http.DefaultClient
	transport  = http.DefaultTransport
	bodyReader = ioutil.ReadAll
)

type manifest struct {
	DirectorUUID  interface{} `yaml:"director_uuid"`
	Name          interface{} `yaml:"name"`
	Compilation   interface{} `yaml:"compilation"`
	Update        interface{} `yaml:"update"`
	Networks      interface{} `yaml:"networks"`
	ResourcePools []struct {
		Name            interface{} `yaml:"name"`
		Network         interface{} `yaml:"network"`
		Size            interface{} `yaml:"size,omitempty"`
		CloudProperties interface{} `yaml:"cloud_properties,omitempty"`
		Env             interface{} `yaml:"env,omitempty"`
		Stemcell        struct {
			Name    string `yaml:"name"`
			Version string `yaml:"version"`
		} `yaml:"stemcell"`
	} `yaml:"resource_pools"`
	Jobs       interface{} `yaml:"jobs"`
	Properties interface{} `yaml:"properties"`
	Releases   []struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	} `yaml:"releases"`
}

type Config struct {
	URL                 string
	Username            string
	Password            string
	TaskPollingInterval time.Duration
	AllowInsecureSSL    bool
}

type Client struct {
	config Config
}

type DirectorInfo struct {
	UUID string
	CPI  string
}

func NewClient(config Config) Client {
	if config.TaskPollingInterval == time.Duration(0) {
		config.TaskPollingInterval = 5 * time.Second
	}

	if config.AllowInsecureSSL {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}

		client = &http.Client{
			Transport: transport,
		}
	}

	return Client{
		config: config,
	}
}

func (c Client) checkTask(location string) error {
	var task struct {
		State  string
		Result string
	}

	for {
		request, err := http.NewRequest("GET", location, nil)
		if err != nil {
			return err
		}
		request.SetBasicAuth(c.config.Username, c.config.Password)

		response, err := transport.RoundTrip(request)
		if err != nil {
			return err
		}

		err = json.NewDecoder(response.Body).Decode(&task)
		if err != nil {
			return err
		}

		switch task.State {
		case "done":
			return nil
		case "error":
			return fmt.Errorf("bosh task failed with an error status %q", task.Result)
		case "errored":
			return fmt.Errorf("bosh task failed with an errored status %q", task.Result)
		case "cancelled":
			return errors.New("bosh task was cancelled")
		default:
			time.Sleep(c.config.TaskPollingInterval)
		}
	}
}

func (c Client) Stemcell(name string) (Stemcell, error) {
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/stemcells", c.config.URL), nil)
	if err != nil {
		return Stemcell{}, err
	}

	request.SetBasicAuth(c.config.Username, c.config.Password)
	response, err := client.Do(request)
	if err != nil {
		return Stemcell{}, err
	}

	if response.StatusCode == http.StatusNotFound {
		return Stemcell{}, fmt.Errorf("stemcell %s could not be found", name)
	}

	if response.StatusCode != http.StatusOK {
		return Stemcell{}, fmt.Errorf("unexpected response %d %s", response.StatusCode, http.StatusText(response.StatusCode))
	}

	stemcells := []struct {
		Name    string
		Version string
	}{}

	err = json.NewDecoder(response.Body).Decode(&stemcells)
	if err != nil {
		return Stemcell{}, err
	}

	stemcell := NewStemcell()
	stemcell.Name = name

	for _, s := range stemcells {
		if s.Name == name {
			stemcell.Versions = append(stemcell.Versions, s.Version)
		}
	}

	return stemcell, nil
}

func (c Client) Release(name string) (Release, error) {
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/releases/%s", c.config.URL, name), nil)
	if err != nil {
		return Release{}, err
	}

	request.SetBasicAuth(c.config.Username, c.config.Password)
	response, err := client.Do(request)
	if err != nil {
		return Release{}, err
	}

	if response.StatusCode == http.StatusNotFound {
		return Release{}, fmt.Errorf("release %s could not be found", name)
	}

	if response.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("unexpected response %d %s", response.StatusCode, http.StatusText(response.StatusCode))
	}

	release := NewRelease()
	err = json.NewDecoder(response.Body).Decode(&release)
	if err != nil {
		return Release{}, err
	}

	release.Name = name

	return release, nil
}

type VM struct {
	State string `json:"job_state"`
}

func (c Client) DeploymentVMs(name string) ([]VM, error) {
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/deployments/%s/vms?format=full", c.config.URL, name), nil)
	if err != nil {
		return []VM{}, err
	}

	request.SetBasicAuth(c.config.Username, c.config.Password)
	response, err := transport.RoundTrip(request)
	if err != nil {
		return []VM{}, err
	}

	if response.StatusCode != http.StatusFound {
		return []VM{}, fmt.Errorf("unexpected response %d %s", response.StatusCode, http.StatusText(response.StatusCode))
	}

	location := response.Header.Get("Location")

	err = c.checkTask(location)
	if err != nil {
		return []VM{}, err
	}

	request, err = http.NewRequest("GET", fmt.Sprintf("%s/output?type=result", location), nil)
	if err != nil {
		return []VM{}, err
	}

	request.SetBasicAuth(c.config.Username, c.config.Password)
	response, err = transport.RoundTrip(request)
	if err != nil {
		return []VM{}, err
	}

	body, err := bodyReader(response.Body)
	if err != nil {
		return []VM{}, err
	}
	defer response.Body.Close()

	body = bytes.TrimSpace(body)
	parts := bytes.Split(body, []byte("\n"))

	var vms []VM
	for _, part := range parts {
		var vm VM
		err = json.Unmarshal(part, &vm)
		if err != nil {
			return vms, err
		}

		vms = append(vms, vm)
	}

	return vms, nil
}

func (c Client) Info() (DirectorInfo, error) {
	response, err := client.Get(fmt.Sprintf("%s/info", c.config.URL))
	if err != nil {
		return DirectorInfo{}, err
	}

	info := DirectorInfo{}
	err = json.NewDecoder(response.Body).Decode(&info)
	if err != nil {
		return DirectorInfo{}, err
	}

	return info, nil
}

func (c Client) Deploy(manifest []byte) error {
	if len(manifest) == 0 {
		return errors.New("a valid manifest is required to deploy")
	}

	request, err := http.NewRequest("POST", fmt.Sprintf("%s/deployments", c.config.URL), bytes.NewBuffer(manifest))
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "text/yaml")
	request.SetBasicAuth(c.config.Username, c.config.Password)

	response, err := transport.RoundTrip(request)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusFound {
		return fmt.Errorf("unexpected response %d %s", response.StatusCode, http.StatusText(response.StatusCode))
	}

	return c.checkTask(response.Header.Get("Location"))
}

func (c Client) ScanAndFix(yaml []byte) error {
	var manifest struct {
		Name string
		Jobs []struct {
			Name      string
			Instances int
		}
	}
	err := candiedyaml.Unmarshal(yaml, &manifest)
	if err != nil {
		return err
	}

	jobs := make(map[string][]int)
	for _, j := range manifest.Jobs {
		if j.Instances > 0 {
			var indices []int
			for i := 0; i < j.Instances; i++ {
				indices = append(indices, i)
			}
			jobs[j.Name] = indices
		}
	}

	requestBody, err := json.Marshal(map[string]interface{}{
		"jobs": jobs,
	})
	if err != nil {
		return err
	}

	request, err := http.NewRequest("PUT", fmt.Sprintf("%s/deployments/%s/scan_and_fix", c.config.URL, manifest.Name), bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	request.SetBasicAuth(c.config.Username, c.config.Password)
	request.Header.Set("Content-Type", "application/json")

	response, err := transport.RoundTrip(request)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusFound {
		return fmt.Errorf("unexpected response %d %s", response.StatusCode, http.StatusText(response.StatusCode))
	}

	err = c.checkTask(response.Header.Get("Location"))
	if err != nil {
		return err
	}

	return nil
}

func (c Client) DeleteDeployment(name string) error {
	if name == "" {
		return errors.New("a valid deployment name is required")
	}

	request, err := http.NewRequest("DELETE", fmt.Sprintf("%s/deployments/%s", c.config.URL, name), nil)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "text/yaml")
	request.SetBasicAuth(c.config.Username, c.config.Password)

	response, err := transport.RoundTrip(request)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusFound {
		return fmt.Errorf("unexpected response %d %s", response.StatusCode, http.StatusText(response.StatusCode))
	}

	return c.checkTask(response.Header.Get("Location"))
}

func (c Client) ResolveManifestVersions(yaml []byte) ([]byte, error) {
	m := manifest{}
	err := candiedyaml.Unmarshal(yaml, &m)
	if err != nil {
		return nil, err
	}

	for i, r := range m.Releases {
		if r.Version == "latest" {
			release, err := c.Release(r.Name)
			if err != nil {
				return nil, err
			}
			r.Version = release.Latest()
			m.Releases[i] = r
		}
	}

	for i, pool := range m.ResourcePools {
		if pool.Stemcell.Version == "latest" {
			stemcell, err := c.Stemcell(pool.Stemcell.Name)
			if err != nil {
				return nil, err
			}
			pool.Stemcell.Version = stemcell.Latest()
			m.ResourcePools[i] = pool
		}
	}

	return candiedyaml.Marshal(m)
}
