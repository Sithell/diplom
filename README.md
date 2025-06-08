# Kubernetes Setup Tool

A production-ready tool for automating the setup of Kubernetes clusters with monitoring capabilities.

## Features

- Automated Kubernetes cluster setup
- Monitoring stack deployment (Prometheus + Grafana)
- System requirements verification
- Backup creation
- Progress tracking and status reporting
- Support for multiple nodes

## Prerequisites

- Go 1.16 or later
- SSH access to target machines
- Root or sudo privileges on target machines

## Installation

```bash
git clone https://github.com/maarulav/k8s-setup.git
cd k8s-setup
go build -o k8s-setup cmd/k8s-setup/main.go
```

## Configuration

Create a configuration file (e.g., `config.json`) with the following structure:

```json
{
  "ssh": {
    "username": "root",
    "password": "your-password",
    "keyFile": "/path/to/private/key",
    "timeout": 30
  },
  "kubernetes": {
    "version": "1.24.0-00",
    "podCIDR": "10.244.0.0/16",
    "serviceCIDR": "10.96.0.0/12"
  },
  "monitoring": {
    "prometheus": {
      "retentionTime": "15d",
      "storageClass": "standard"
    },
    "grafana": {
      "adminPassword": "your-secure-password",
      "domain": "grafana.example.com"
    }
  },
  "resources": {
    "cpu": "2",
    "memory": "4Gi"
  }
}
```

## Usage

```bash
./k8s-setup config.json <ip1> <ip2> <ip3>
```

Where:
- `config.json` is the path to your configuration file
- `<ip1>`, `<ip2>`, `<ip3>` are the IP addresses of the target machines

## Project Structure

```
.
├── cmd/
│   └── k8s-setup/
│       └── main.go
├── pkg/
│   ├── config/
│   │   └── config.go
│   ├── ssh/
│   │   └── ssh.go
│   ├── kubernetes/
│   │   └── kubernetes.go
│   ├── monitoring/
│   │   └── monitoring.go
│   └── backup/
│       └── backup.go
├── internal/
│   ├── logger/
│   │   └── logger.go
│   └── status/
│       └── status.go
├── go.mod
├── go.sum
└── README.md
```

## Status Tracking

The tool creates a `status` directory containing JSON files for each VM being set up. These files track the progress and any errors that occur during the setup process.

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details. # diplom
