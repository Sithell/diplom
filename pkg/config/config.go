package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"
)

// Config represents the application configuration
type Config struct {
	SSHConfig struct {
		Username string `json:"username"`
		Password string `json:"password"`
		KeyFile  string `json:"keyFile"`
		Timeout  int    `json:"timeout"`
	} `json:"ssh"`
	Kubernetes struct {
		Version     string `json:"version"`
		PodCIDR     string `json:"podCIDR"`
		ServiceCIDR string `json:"serviceCIDR"`
	} `json:"kubernetes"`
	Monitoring struct {
		Prometheus struct {
			RetentionTime string `json:"retentionTime"`
			StorageClass  string `json:"storageClass"`
		} `json:"prometheus"`
		Grafana struct {
			AdminPassword string `json:"adminPassword"`
			Domain        string `json:"domain"`
		} `json:"grafana"`
	} `json:"monitoring"`
	Resources struct {
		CPU    string `json:"cpu"`
		Memory string `json:"memory"`
	} `json:"resources"`
}

// VMConfig represents configuration for a single VM
type VMConfig struct {
	IP       string
	Username string
	Password string
	KeyFile  string
	Timeout  time.Duration
}

// LoadConfig loads configuration from a JSON file
func LoadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return &config, nil
}
