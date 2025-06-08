package kubernetes

import (
	"fmt"
	"time"

	"github.com/maarulav/k8s-setup/pkg/config"
	"github.com/maarulav/k8s-setup/pkg/ssh"
)

// Setup sets up Kubernetes on the remote server
func Setup(client *ssh.Client, config *config.Config) error {
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
		output, err := client.ExecuteCommand(cmd)
		if err != nil {
			return fmt.Errorf("failed to execute command '%s': %v\nOutput: %s", cmd, err, output)
		}
		time.Sleep(2 * time.Second)
	}

	return nil
}

// Verify verifies the Kubernetes setup
func Verify(client *ssh.Client) error {
	commands := []string{
		"kubectl get nodes",
		"kubectl get pods -A",
		"kubectl get services -A",
		"kubectl get deployments -A",
	}

	for _, cmd := range commands {
		output, err := client.ExecuteCommand(cmd)
		if err != nil {
			return fmt.Errorf("verification failed for command '%s': %v", cmd, err)
		}
		fmt.Printf("Verification output for %s:\n%s", cmd, output)
	}

	return nil
}
