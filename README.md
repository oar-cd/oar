# Oar

GitOps for Docker Compose. Think ArgoCD, but with the simplicity of compose.yaml instead of Kubernetes complexity.

## What is Oar?

Oar brings GitOps principles to Docker Compose deployments. It automatically synchronizes your Git repositories with running Docker Compose applications, providing declarative infrastructure management without the overhead of Kubernetes.

Your Git repository becomes the single source of truth for your application deployments. When you push changes to your compose.yaml files, Oar detects the changes and automatically updates your running services.

## Features

- **GitOps Workflow**: Declarative deployments driven by Git repositories
- **Automatic Synchronization**: Continuously monitors Git repositories for changes
- **Compose File Discovery**: Automatically discovers and manages compose.yaml files
- **Real-time Deployment**: Live deployment status and logs via Server-Sent Events
- **Secure Git Integration**: Encrypted credential storage for private repositories
- **Web Interface**: Modern UI for managing projects and monitoring deployments
- **Simple Architecture**: All the benefits of GitOps without Kubernetes complexity

## Who is it for?

Oar is perfect for teams and individuals who:
- Want GitOps benefits without Kubernetes overhead
- Prefer Docker Compose simplicity over complex orchestration
- Need centralized management of multiple Compose applications
- Want automated deployments triggered by Git commits
- Require self-hosted deployment management solutions

## Installation

```bash
curl -sSL https://raw.githubusercontent.com/ch00k/oar/main/install.sh | bash
```

The web interface will be available at `http://localhost:3333`.

## Development

For development, clone the repository and use the provided Makefile:

```bash
git clone https://github.com/ch00k/oar.git
cd oar

# Start development environment with hot reloading
make dev

# Run tests
make test

# Run linting
make lint
```

## License

This project is open source. See the repository for license details.
