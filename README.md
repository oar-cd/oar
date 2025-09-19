<p align="center">
  <img src="web/assets/icons/logo.svg" alt="Oar Logo" width="196" height="196">
</p>
<br>

[![CI](https://github.com/oar-cd/oar/actions/workflows/ci.yml/badge.svg)](https://github.com/oar-cd/oar/actions/workflows/ci.yml)&nbsp;
[![codecov](https://codecov.io/gh/oar-cd/oar/graph/badge.svg?token=N1Dyy2nFt5)](https://codecov.io/gh/oar-cd/oar)&nbsp;
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/oar-cd/oar)](https://github.com/oar-cd/oar/releases/latest)

# Oar

Self-hosted Docker Compose project management with GitOps workflows. All the benefits of declarative deployments without Kubernetes complexity.

Oar bridges the gap between simple Docker Compose deployments and complex Kubernetes orchestration. Perfect for teams running containerized applications on a single server or small cluster who want GitOps automation without operational overhead.

## What is Oar?

Oar is a lightweight, self-hosted platform that automates Docker Compose deployments using GitOps principles. It monitors your Git repositories and automatically deploys changes to your running services, providing the reliability of declarative infrastructure with the simplicity of Docker Compose.

**Perfect for:**
- Development and staging environments
- Small to medium production workloads
- Teams transitioning from manual deployments to GitOps
- Organizations wanting Kubernetes-style automation without complexity
- Side projects and personal infrastructure

## Key Features

### GitOps Automation
- **Continuous Deployment**: Automatic deployments triggered by Git commits
- **Drift Detection**: Ensures running services match Git repository state
- **Rollback Support**: Easy rollbacks to previous Git commits
- **Multi-Repository Support**: Manage multiple projects from different repositories

### Developer Experience
- **Web Interface**: Modern, responsive UI for managing all deployments
- **Real-time Streaming**: Live deployment logs and service status updates
- **CLI Integration**: Full command-line interface for automation and scripting
- **Zero Configuration**: Works with existing Docker Compose files

### Security & Operations
- **Encrypted Credentials**: Secure storage of Git authentication tokens
- **Private Repository Support**: Works with GitHub, GitLab, Bitbucket, and self-hosted Git
- **Service Discovery**: Automatic detection of Docker Compose files in repositories
- **Resource Monitoring**: Track service health and resource usage

## Why Choose Oar Over Alternatives?

| Feature | Oar | Portainer | Kubernetes + ArgoCD | Manual Docker Compose |
|---------|-----|-----------|---------------------|----------------------|
| **Setup Complexity** | Minimal | Low | High | None |
| **GitOps Automation** | ✅ Native | ❌ | ✅ Complex | ❌ |
| **Resource Overhead** | Low | Low | High | Minimal |
| **Learning Curve** | Gentle | Moderate | Steep | None |
| **Multi-Environment** | ✅ | ✅ | ✅ | Manual |
| **Declarative Config** | ✅ | ❌ | ✅ | ✅ |
| **Auto-deployment** | ✅ | ❌ | ✅ | ❌ |

**Oar vs Portainer**: Oar focuses on GitOps automation while Portainer is primarily a container management UI
**Oar vs Kubernetes**: Oar provides similar declarative benefits with 90% less operational complexity
**Oar vs Manual**: Eliminates deployment errors and configuration drift with minimal overhead

## Quick Start

### Installation

```bash
curl -sSL https://github.com/oar-cd/oar/releases/latest/download/install.sh | bash
```

### Basic Usage

1. **Start the server**:
   ```bash
   oar server start
   ```

2. **Open the web interface**: Navigate to `http://localhost:3333`

3. **Create your first project**:
   - Click "New Project"
   - Enter your Git repository URL
   - Oar will automatically discover Docker Compose files
   - Configure deployment settings
   - Deploy!

### Example: Deploy a Web Application

```bash
# Create a new project from command line
oar project create my-app https://github.com/username/my-docker-app.git

# Deploy the project
oar project deploy my-app

# View deployment status
oar project status my-app

# Stream live logs
oar project logs my-app
```

## How It Works

1. **Repository Setup**: Point Oar to your Git repository containing Docker Compose files
2. **Automatic Discovery**: Oar scans for `docker-compose.yml` or `compose.yml` files
3. **Deployment**: Choose compose file and deployment configuration
4. **GitOps Monitoring**: Oar watches for new commits and automatically redeploys
5. **Management**: Use web UI or CLI to manage services, view logs, and monitor health

## Use Cases

### Development Teams
- **Staging Environments**: Automatically deploy feature branches for testing
- **Integration Testing**: Consistent environments that match production
- **Developer Onboarding**: New team members get identical setups instantly

### Small Production Workloads
- **Microservices**: Deploy and manage multiple related services
- **Web Applications**: Full-stack applications with databases and caching
- **APIs and Backends**: RESTful services with automatic scaling

### Personal Projects
- **Home Labs**: Self-hosted applications and services
- **Side Projects**: Quick deployment without infrastructure complexity
- **Learning**: Practice DevOps without Kubernetes overhead

## Requirements

- Docker and Docker Compose installed
- Git repositories with Docker Compose files
- Linux, macOS, or Windows with WSL2
- Minimum 1GB RAM, 1 CPU core

## Screenshots & Demo

> **Coming Soon**: Screenshots and demo videos showing the web interface and deployment workflows

## Configuration

Oar works out of the box with minimal configuration. Advanced options include:

- **Data Directory**: Customize where projects and data are stored
- **Port Configuration**: Change default web interface port (3333)
- **Git Credentials**: Configure authentication for private repositories
- **Deployment Hooks**: Custom scripts before/after deployments
- **Environment Variables**: Pass secrets and configuration to containers

See the [Configuration Guide](docs/configuration.md) for detailed setup options.

## Architecture

Oar consists of:
- **CLI**: Command-line interface for project management
- **Web Server**: HTTP API and web interface
- **Watcher Service**: GitOps monitoring and automatic deployments
- **Database**: SQLite for project configuration and state
- **Git Integration**: Repository cloning and change detection

## Development

### Prerequisites
- Go 1.21+
- Node.js (for TailwindCSS)
- Make

### Development Setup

```bash
# Clone the repository
git clone https://github.com/oar-cd/oar.git
cd oar

# Start development environment with hot reloading
make dev

# Or run individual components
make generate  # Build CSS and templates
make air       # Start server with hot reloading
make test      # Run tests
make lint      # Run linting
```

### Project Structure
- `cmd/` - CLI commands and application entry points
- `web/` - Web server, templates, and assets
- `services/` - Business logic for Git, Docker Compose, and project management
- `models/` - Database models and migrations
- `watcher/` - GitOps monitoring service

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Quick Contribution Steps
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run `make test` and `make lint`
6. Submit a pull request

## Support

- **Documentation**: [docs.oar-cd.dev](https://docs.oar-cd.dev)
- **Issues**: [GitHub Issues](https://github.com/oar-cd/oar/issues)
- **Discussions**: [GitHub Discussions](https://github.com/oar-cd/oar/discussions)
- **Community**: [Discord](https://discord.gg/oar-cd)

## License

MIT License. See [LICENSE](LICENSE) for details.

## Roadmap

- **v1.1**: Multi-environment support (dev/staging/prod)
- **v1.2**: Advanced deployment strategies (blue-green, rolling)
- **v1.3**: Kubernetes deployment support
- **v1.4**: Team collaboration features
- **v1.5**: Monitoring and alerting integration

---

**Ready to get started?** Install Oar and deploy your first project in under 5 minutes!
