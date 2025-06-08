package ssh

import (
	"fmt"
	"io/ioutil"

	"github.com/maarulav/k8s-setup/pkg/config"
	"golang.org/x/crypto/ssh"
)

// Client represents an SSH client
type Client struct {
	*ssh.Client
}

// Connect establishes an SSH connection
func Connect(config config.VMConfig) (*Client, error) {
	sshConfig := &ssh.ClientConfig{
		User: config.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(config.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         config.Timeout,
	}

	if config.KeyFile != "" {
		key, err := ioutil.ReadFile(config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read key file: %v", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %v", err)
		}

		sshConfig.Auth = []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		}
	}

	client, err := ssh.Dial("tcp", config.IP+":22", sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %v", err)
	}

	return &Client{client}, nil
}

// ExecuteCommand executes a command on the remote server
func (c *Client) ExecuteCommand(command string) (string, error) {
	session, err := c.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output), fmt.Errorf("command failed: %v", err)
	}

	return string(output), nil
}

// CheckSystemRequirements checks if the system meets the requirements
func (c *Client) CheckSystemRequirements() error {
	commands := []string{
		"uname -a",
		"free -h",
		"df -h",
		"nproc",
		"cat /etc/os-release",
	}

	for _, cmd := range commands {
		output, err := c.ExecuteCommand(cmd)
		if err != nil {
			return fmt.Errorf("system check failed: %v", err)
		}
		fmt.Printf("System check output for %s:\n%s", cmd, output)
	}

	return nil
}
