package monitoring

import (
	"fmt"
	"io/ioutil"

	"github.com/maarulav/k8s-setup/pkg/config"
	"github.com/maarulav/k8s-setup/pkg/ssh"
)

// Setup sets up monitoring stack on the remote server
func Setup(client *ssh.Client, config *config.Config) error {
	// Create monitoring namespace
	if _, err := client.ExecuteCommand("kubectl create namespace monitoring"); err != nil {
		return fmt.Errorf("failed to create monitoring namespace: %v", err)
	}

	// Install Helm
	helmCommands := []string{
		"curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash",
		"helm repo add prometheus-community https://prometheus-community.github.io/helm-charts",
		"helm repo update",
	}

	for _, cmd := range helmCommands {
		if _, err := client.ExecuteCommand(cmd); err != nil {
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
	if _, err := client.ExecuteCommand(installCmd); err != nil {
		return fmt.Errorf("failed to install Prometheus stack: %v", err)
	}

	// Configure Grafana
	grafanaCommands := []string{
		fmt.Sprintf("kubectl create secret generic grafana-admin --from-literal=admin-password=%s -n monitoring", config.Monitoring.Grafana.AdminPassword),
		"kubectl patch deployment prometheus-grafana -n monitoring --type=json -p='[{\"op\": \"add\", \"path\": \"/spec/template/spec/containers/0/env/0\", \"value\": {\"name\": \"GF_SECURITY_ADMIN_PASSWORD\", \"valueFrom\": {\"secretKeyRef\": {\"name\": \"grafana-admin\", \"key\": \"admin-password\"}}}}]'",
	}

	for _, cmd := range grafanaCommands {
		if _, err := client.ExecuteCommand(cmd); err != nil {
			return fmt.Errorf("failed to configure Grafana: %v", err)
		}
	}

	// Wait for pods to be ready
	waitCommands := []string{
		"kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=prometheus -n monitoring --timeout=300s",
		"kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=grafana -n monitoring --timeout=300s",
	}

	for _, cmd := range waitCommands {
		if _, err := client.ExecuteCommand(cmd); err != nil {
			return fmt.Errorf("failed to wait for pods: %v", err)
		}
	}

	return nil
}
