<p align="center">
  <img src="web/assets/icons/logo.svg" alt="Oar Logo" width="196" height="196">
</p>
<br>

[![CI](https://github.com/oar-cd/oar/actions/workflows/ci.yml/badge.svg)](https://github.com/oar-cd/oar/actions/workflows/ci.yml)&nbsp;
[![codecov](https://codecov.io/gh/oar-cd/oar/graph/badge.svg?token=N1Dyy2nFt5)](https://codecov.io/gh/oar-cd/oar)&nbsp;
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/oar-cd/oar)](https://github.com/oar-cd/oar/releases/latest)

# Oar

GitOps automation for Docker Compose on a single Docker host. Oar watches a Git branch, runs `docker compose` to apply updates, and records deployment results for review in the CLI or web UI.

## Overview

- Single binary exposing the CLI and web dashboard.
- Tracks project configuration on disk and can encrypt stored secrets when an encryption key is provided.
- Automatically reconciles Docker Compose projects without manual intervention.

## Quick Start

#### Prerequisites

- Docker Engine with the Docker Compose plugin on a Linux host.
- Network access from the host to your Git remote.

#### Installation, upgrade

> [!IMPORTANT]
> `sudo` is required

```bash
curl -sSL https://github.com/oar-cd/oar/releases/latest/download/install.sh | bash
```

Access the web UI at *http://127.0.0.1:4777*

## Target Audience & Use Cases

Oar is designed as *ArgoCD for Docker Compose* - bringing GitOps automation to environments where Kubernetes complexity isn't needed or justified.

### Ideal for

- *Home labs and personal projects* - Simple single-server deployments without operational overhead
- *Demo environments* - Fast setup and teardown for rapid prototyping and demonstrations
- *Development and staging environments* - Quick deployment cycles without production-grade complexity
- *Small-scale applications* - Projects that don't require multi-node orchestration or enterprise features
- *Learning environments* - Educational setups where Docker Compose is more approachable than Kubernetes
- *Side projects and experiments* - Personal or small team projects where simplicity trumps scalability

### When to choose Oar over alternatives

- You need GitOps but not the complexity of Kubernetes
- Downtime during deployments is acceptable (rolling updates not required)
- You prefer Docker Compose's familiar syntax over Kubernetes manifests
- You want automated deployments without managing a full cluster
- You need something that "just works" on a single host

## How Oar Compares

|                               | Oar | ArgoCD | Portainer | DIY Scripts |
|-------------------------------|-----|--------|-----------|-------------|
| *Single host deployment*      | ✓   | ✗      | ✓         | ✓           |
| *GitOps automation*           | ✓   | ✓      | ✗         | ✗           |
| *Built-in secrets encryption* | ✓   | ✗      | ✗         | ✗           |
| *Web dashboard*               | ✓   | ✓      | ✓         | ✗           |
| *Docker Compose native*       | ✓   | ✗      | ✓         | ✓           |
| *Zero-config setup*           | ✓   | ✗      | ✗         | ✗           |
| *Deployment history*          | ✓   | ✓      | ✗         | ✗           |
| *Auto drift detection*        | ✓   | ✓      | ✗         | ✗           |
