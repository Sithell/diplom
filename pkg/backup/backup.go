package backup

import (
	"fmt"

	"github.com/maarulav/k8s-setup/pkg/ssh"
)

// Create creates a backup of the Kubernetes cluster
func Create(client *ssh.Client) error {
	backupDir := "/root/k8s-backup"
	commands := []string{
		fmt.Sprintf("mkdir -p %s", backupDir),
		fmt.Sprintf("kubectl get all -A -o yaml > %s/all-resources.yaml", backupDir),
		fmt.Sprintf("kubectl get configmaps -A -o yaml > %s/configmaps.yaml", backupDir),
		fmt.Sprintf("kubectl get secrets -A -o yaml > %s/secrets.yaml", backupDir),
		fmt.Sprintf("tar -czf %s/k8s-backup.tar.gz %s", backupDir, backupDir),
	}

	for _, cmd := range commands {
		if _, err := client.ExecuteCommand(cmd); err != nil {
			return fmt.Errorf("backup failed: %v", err)
		}
	}

	return nil
}
