package destiny

import "github.com/cloudfoundry-incubator/candiedyaml"

type Compilation struct {
	CloudProperties     CompilationCloudProperties `yaml:"cloud_properties"`
	Network             string                     `yaml:"network"`
	ReuseCompilationVMs bool                       `yaml:"reuse_compilation_vms"`
	Workers             int                        `yaml:"workers"`
}

type CompilationCloudProperties struct {
	InstanceType     string                                   `yaml:"instance_type,omitempty"`
	AvailabilityZone string                                   `yaml:"availability_zone,omitempty"`
	EphemeralDisk    *CompilationCloudPropertiesEphemeralDisk `yaml:"ephemeral_disk,omitempty"`
}

type CompilationCloudPropertiesEphemeralDisk struct {
	Size int    `yaml:"size"`
	Type string `yaml:"type"`
}

type Release struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type ResourcePool struct {
	CloudProperties ResourcePoolCloudProperties `yaml:"cloud_properties"`
	Name            string                      `yaml:"name"`
	Network         string                      `yaml:"network"`
	Stemcell        ResourcePoolStemcell        `yaml:"stemcell"`
}

type ResourcePoolCloudProperties struct {
	InstanceType     string                                    `yaml:"instance_type,omitempty"`
	AvailabilityZone string                                    `yaml:"availability_zone,omitempty"`
	EphemeralDisk    *ResourcePoolCloudPropertiesEphemeralDisk `yaml:"ephemeral_disk,omitempty"`
}

type ResourcePoolCloudPropertiesEphemeralDisk struct {
	Size int    `yaml:"size"`
	Type string `yaml:"type"`
}

type ResourcePoolStemcell struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type Update struct {
	Canaries        int    `yaml:"canaries"`
	CanaryWatchTime string `yaml:"canary_watch_time"`
	MaxInFlight     int    `yaml:"max_in_flight"`
	Serial          bool   `yaml:"serial"`
	UpdateWatchTime string `yaml:"update_watch_time"`
}

type Manifest struct {
	Compilation   Compilation    `yaml:"compilation"`
	DirectorUUID  string         `yaml:"director_uuid"`
	Jobs          []Job          `yaml:"jobs"`
	Name          string         `yaml:"name"`
	Networks      []Network      `yaml:"networks"`
	Properties    Properties     `yaml:"properties"`
	Releases      []Release      `yaml:"releases"`
	ResourcePools []ResourcePool `yaml:"resource_pools"`
	Update        Update         `yaml:"update"`
}

func (m Manifest) ToYAML() ([]byte, error) {
	return candiedyaml.Marshal(m)
}

func FromYAML(yaml []byte) (Manifest, error) {
	var m Manifest
	if err := candiedyaml.Unmarshal(yaml, &m); err != nil {
		return m, err
	}
	return m, nil
}

func StemcellForIAAS(iaas int) string {
	switch iaas {
	case AWS:
		return AWSStemcell
	case Warden:
		return WardenStemcell
	default:
		panic("failed to find a stemcell for the given IAAS")
	}
}
