package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
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

// SetupStatus tracks the progress of setup
type SetupStatus struct {
	VMIP           string    `json:"vmIP"`
	StartTime      time.Time `json:"startTime"`
	EndTime        time.Time `json:"endTime"`
	CurrentStep    string    `json:"currentStep"`
	Status         string    `json:"status"`
	Error          string    `json:"error,omitempty"`
	CompletedSteps []string  `json:"completedSteps"`
}

// Logger provides structured logging
type Logger struct {
	*log.Logger
	status *SetupStatus
}

func main() {
	// Initialize logger
	logger := &Logger{
		Logger: log.New(os.Stdout, "", log.LstdFlags),
	}

	// Parse command line arguments
	if len(os.Args) < 2 {
		logger.Fatal("Usage: ./k8s-setup <config.json> <ip1> <ip2> <ip3> ...")
	}

	// Load configuration
	config, err := loadConfig(os.Args[1])
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Get IP addresses from command line arguments
	ips := os.Args[2:]

	// Create status directory
	if err := os.MkdirAll("status", 0755); err != nil {
		logger.Fatalf("Failed to create status directory: %v", err)
	}

	// Process each VM
	for _, ip := range ips {
		status := &SetupStatus{
			VMIP:        ip,
			StartTime:   time.Now(),
			CurrentStep: "Initializing",
			Status:      "In Progress",
		}

		logger.status = status
		logger.Printf("Starting setup for VM %s", ip)

		// Create VM configuration
		vmConfig := VMConfig{
			IP:       ip,
			Username: config.SSHConfig.Username,
			Password: config.SSHConfig.Password,
			KeyFile:  config.SSHConfig.KeyFile,
			Timeout:  time.Duration(config.SSHConfig.Timeout) * time.Second,
		}

		// Connect to VM
		client, err := connectSSH(vmConfig)
		if err != nil {
			status.Status = "Failed"
			status.Error = fmt.Sprintf("SSH connection failed: %v", err)
			saveStatus(status)
			continue
		}
		defer client.Close()

		// Check system requirements
		if err := checkSystemRequirements(client); err != nil {
			status.Status = "Failed"
			status.Error = fmt.Sprintf("System requirements check failed: %v", err)
			saveStatus(status)
			continue
		}

		// Setup Kubernetes
		status.CurrentStep = "Setting up Kubernetes"
		if err := setupKubernetes(client, config); err != nil {
			status.Status = "Failed"
			status.Error = fmt.Sprintf("Kubernetes setup failed: %v", err)
			saveStatus(status)
			continue
		}
		status.CompletedSteps = append(status.CompletedSteps, "kubernetes")

		// Setup monitoring
		status.CurrentStep = "Setting up monitoring"
		if err := setupMonitoring(client, config); err != nil {
			status.Status = "Failed"
			status.Error = fmt.Sprintf("Monitoring setup failed: %v", err)
			saveStatus(status)
			continue
		}
		status.CompletedSteps = append(status.CompletedSteps, "monitoring")

		// Verify setup
		status.CurrentStep = "Verifying setup"
		if err := verifySetup(client); err != nil {
			status.Status = "Failed"
			status.Error = fmt.Sprintf("Verification failed: %v", err)
			saveStatus(status)
			continue
		}
		status.CompletedSteps = append(status.CompletedSteps, "verification")

		// Create backup
		status.CurrentStep = "Creating backup"
		if err := createBackup(client); err != nil {
			logger.Printf("Warning: Backup creation failed: %v", err)
		} else {
			status.CompletedSteps = append(status.CompletedSteps, "backup")
		}

		status.Status = "Completed"
		status.EndTime = time.Now()
		saveStatus(status)
		logger.Printf("Setup completed successfully for VM %s", ip)
	}
}

func loadConfig(path string) (*Config, error) {
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

func connectSSH(config VMConfig) (*ssh.Client, error) {
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

	return ssh.Dial("tcp", config.IP+":22", sshConfig)
}

func executeCommand(client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
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

func checkSystemRequirements(client *ssh.Client) error {
	commands := []string{
		"uname -a",
		"free -h",
		"df -h",
		"nproc",
		"cat /etc/os-release",
	}

	for _, cmd := range commands {
		output, err := executeCommand(client, cmd)
		if err != nil {
			return fmt.Errorf("system check failed: %v", err)
		}
		log.Printf("System check output for %s:\n%s", cmd, output)
	}

	return nil
}

func setupKubernetes(client *ssh.Client, config *Config) error {
	commands := []string{
		// Update system
		"apt-get update && apt-get upgrade -y",

		// Install required packages
		"apt-get install -y apt-transport-https ca-certificates curl software-properties-common",

		// Add Docker repository
		"curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -",
		"add-apt-repository \"deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable\"",

		// Install Docker
		"apt-get update && apt-get install -y docker-ce docker-ce-cli containerd.io",

		// Configure Docker
		"mkdir -p /etc/docker",
		`cat > /etc/docker/daemon.json << EOF
{
  "exec-opts": ["native.cgroupdriver=systemd"],
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "100m"
  },
  "storage-driver": "overlay2"
}
EOF`,
		"systemctl daemon-reload",
		"systemctl restart docker",

		// Add Kubernetes repository
		"curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -",
		"echo \"deb https://apt.kubernetes.io/ kubernetes-xenial main\" > /etc/apt/sources.list.d/kubernetes.list",

		// Install Kubernetes components
		fmt.Sprintf("apt-get update && apt-get install -y kubelet=%s kubeadm=%s kubectl=%s",
			config.Kubernetes.Version,
			config.Kubernetes.Version,
			config.Kubernetes.Version),

		// Initialize Kubernetes cluster
		fmt.Sprintf("kubeadm init --pod-network-cidr=%s --service-cidr=%s",
			config.Kubernetes.PodCIDR,
			config.Kubernetes.ServiceCIDR),

		// Setup kubectl for root user
		"mkdir -p $HOME/.kube && cp -i /etc/kubernetes/admin.conf $HOME/.kube/config && chown $(id -u):$(id -g) $HOME/.kube/config",

		// Install Calico network plugin
		"kubectl apply -f https://docs.projectcalico.org/manifests/calico.yaml",
	}

	for _, cmd := range commands {
		output, err := executeCommand(client, cmd)
		if err != nil {
			return fmt.Errorf("failed to execute command '%s': %v\nOutput: %s", cmd, err, output)
		}
		time.Sleep(2 * time.Second)
	}

	return nil
}

func setupMonitoring(client *ssh.Client, config *Config) error {
	// Create monitoring namespace
	if _, err := executeCommand(client, "kubectl create namespace monitoring"); err != nil {
		return fmt.Errorf("failed to create monitoring namespace: %v", err)
	}

	// Install Helm
	helmCommands := []string{
		"curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash",
		"helm repo add prometheus-community https://prometheus-community.github.io/helm-charts",
		"helm repo update",
	}

	for _, cmd := range helmCommands {
		if _, err := executeCommand(client, cmd); err != nil {
			return fmt.Errorf("failed to setup Helm: %v", err)
		}
	}

	// Create Prometheus values file
	prometheusValues := fmt.Sprintf(`
prometheus:
  prometheusSpec:
    retention: %s
    storageSpec:
      volumeClaimTemplate:
        spec:
          storageClassName: %s
          accessModes: ["ReadWriteOnce"]
          resources:
            requests:
              storage: 10Gi
`, config.Monitoring.Prometheus.RetentionTime, config.Monitoring.Prometheus.StorageClass)

	if err := ioutil.WriteFile("prometheus-values.yaml", []byte(prometheusValues), 0644); err != nil {
		return fmt.Errorf("failed to create Prometheus values file: %v", err)
	}

	// Install Prometheus stack
	installCmd := fmt.Sprintf("helm install prometheus prometheus-community/kube-prometheus-stack -f prometheus-values.yaml --namespace monitoring")
	if _, err := executeCommand(client, installCmd); err != nil {
		return fmt.Errorf("failed to install Prometheus stack: %v", err)
	}

	// Configure Grafana
	grafanaCommands := []string{
		fmt.Sprintf("kubectl create secret generic grafana-admin --from-literal=admin-password=%s -n monitoring", config.Monitoring.Grafana.AdminPassword),
		"kubectl patch deployment prometheus-grafana -n monitoring --type=json -p='[{\"op\": \"add\", \"path\": \"/spec/template/spec/containers/0/env/0\", \"value\": {\"name\": \"GF_SECURITY_ADMIN_PASSWORD\", \"valueFrom\": {\"secretKeyRef\": {\"name\": \"grafana-admin\", \"key\": \"admin-password\"}}}}]'",
	}

	for _, cmd := range grafanaCommands {
		if _, err := executeCommand(client, cmd); err != nil {
			return fmt.Errorf("failed to configure Grafana: %v", err)
		}
	}

	// Wait for pods to be ready
	waitCommands := []string{
		"kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=prometheus -n monitoring --timeout=300s",
		"kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=grafana -n monitoring --timeout=300s",
	}

	for _, cmd := range waitCommands {
		if _, err := executeCommand(client, cmd); err != nil {
			return fmt.Errorf("failed to wait for pods: %v", err)
		}
	}

	return nil
}

func verifySetup(client *ssh.Client) error {
	commands := []string{
		"kubectl get nodes",
		"kubectl get pods -A",
		"kubectl get services -A",
		"kubectl get deployments -A",
	}

	for _, cmd := range commands {
		output, err := executeCommand(client, cmd)
		if err != nil {
			return fmt.Errorf("verification failed for command '%s': %v", cmd, err)
		}
		log.Printf("Verification output for %s:\n%s", cmd, output)
	}

	return nil
}

func createBackup(client *ssh.Client) error {
	backupDir := "/root/k8s-backup"
	commands := []string{
		fmt.Sprintf("mkdir -p %s", backupDir),
		fmt.Sprintf("kubectl get all -A -o yaml > %s/all-resources.yaml", backupDir),
		fmt.Sprintf("kubectl get configmaps -A -o yaml > %s/configmaps.yaml", backupDir),
		fmt.Sprintf("kubectl get secrets -A -o yaml > %s/secrets.yaml", backupDir),
		fmt.Sprintf("tar -czf %s/k8s-backup.tar.gz %s", backupDir, backupDir),
	}

	for _, cmd := range commands {
		if _, err := executeCommand(client, cmd); err != nil {
			return fmt.Errorf("backup failed: %v", err)
		}
	}

	return nil
}

func saveStatus(status *SetupStatus) error {
	filename := filepath.Join("status", fmt.Sprintf("%s.json", status.VMIP))
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal status: %v", err)
	}

	return ioutil.WriteFile(filename, data, 0644)
}
