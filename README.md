<p align="center">
  <img src="web/assets/icons/logo.svg" alt="Oar Logo" width="196" height="196">
</p>
<br>

[![CI](https://github.com/oar-cd/oar/actions/workflows/ci.yml/badge.svg)](https://github.com/oar-cd/oar/actions/workflows/ci.yml)&nbsp;
[![codecov](https://codecov.io/gh/oar-cd/oar/graph/badge.svg?token=N1Dyy2nFt5)](https://codecov.io/gh/oar-cd/oar)&nbsp;
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/oar-cd/oar)](https://github.com/oar-cd/oar/releases/latest)

# Oar

Self-hosted Docker Compose project management with GitOps workflows. All the benefits of declarative deployments without Kubernetes complexity.

## Why Oar?

Oar brings the power of GitOps to your Docker Compose projects. It allows you to manage your applications from a Git repository, automating deployments and eliminating configuration drift. With Oar, you can turn your Git repositories into the single source of truth for your Docker Compose deployments. Push a change to your repository, and Oar automatically updates your running services.

- **GitOps Made Simple**: ArgoCD-style automation for Docker Compose.
- **Zero Configuration Drift**: Git commits automatically trigger deployments, ensuring your running applications are always in sync with your Git repository.
- **Self-Hosted**: Oar is self-hosted, giving you complete control over your deployment infrastructure.
- **Web UI and CLI**: Manage your projects and deployments through a user-friendly web interface or a powerful command-line interface.
- **Works with Existing Compose Files**: No need to change your existing Docker Compose files.

## Key Features

- **Automated Deployments**: Oar automatically deploys your Docker Compose applications when you push changes to your Git repository.
- **Web UI**: A user-friendly web interface for managing your projects, viewing deployment history, and checking application logs.
- **CLI**: A powerful command-line interface for scripting and automation.
- **Git Integration**: Oar integrates with any Git repository.
- **Docker Compose Support**: Oar works with any Docker Compose file.
- **Notifications**: Oar can notify you of deployment status via a variety of channels (coming soon).

## How It Works

Oar is a Go application that consists of a server and a CLI. The server is responsible for managing your projects, interacting with Git repositories, and deploying Docker Compose applications. The CLI allows you to interact with the Oar server from the command line.

Oar uses a SQLite database to store project information, including the project's name, Git repository URL, and deployment history. It uses the go-git library to interact with Git repositories and the official Docker SDK for Go to interact with the Docker daemon.

## Getting Started

### Installation

You can install Oar using the following command:

```bash
curl -sSL https://github.com/oar-cd/oar/releases/latest/download/install.sh | bash
```

### Running the Server

To start the Oar server, run the following command:

```bash
oar server
```

### Adding a Project

To add a new project, you can use the `oar project add` command:

```bash
oar project add --name my-project --repo-url <your-git-repo-url>
```

## Usage

### CLI

Oar provides a powerful CLI for managing your projects and deployments. Here are some of the most common commands:

- `oar project list`: List all projects.
- `oar project show --name my-project`: Show details for a specific project.
- `oar project deploy --name my-project`: Trigger a new deployment for a project.
- `oar project logs --name my-project`: View the logs for a project.

### Web UI

Oar also provides a web UI for managing your projects. The web UI is available at `http://localhost:8080` by default.

## Comparison with Other Tools

### Oar vs. CI/CD Pipelines (GitHub Actions, GitLab CI, Jenkins)

CI/CD pipelines are great for automating the build and test process, but they are not designed for continuous deployment of Docker Compose applications. Oar is designed specifically for this purpose. It provides a level of automation and control that is not possible with a traditional CI/CD pipeline.

### Oar vs. Docker Swarm and Kubernetes

Docker Swarm and Kubernetes are powerful container orchestration platforms that are designed for large-scale deployments. Oar is designed for smaller-scale deployments where the complexity of Kubernetes is not required. If you are looking for a simple way to automate the deployment of your Docker Compose applications, Oar is a great choice.

## Contributing

We welcome contributions to Oar! Please see the `GEMINI.md` file for information on how to contribute to the project.

## License

Oar is licensed under the MIT License. See the `LICENSE` file for more information.
