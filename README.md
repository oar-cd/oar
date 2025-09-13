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

Turn your Git repositories into the single source of truth for Docker Compose deployments. Push to Git, and Oar automatically syncs your running services - no manual deployments, no configuration drift.

- **GitOps Made Simple**: ArgoCD-style automation for Docker Compose
- **Zero Configuration Drift**: Git commits automatically trigger deployments
- **Self-Hosted**: Complete control over your deployment infrastructure
- **Zero Setup**: Works with existing Compose files

## Installation

```bash
curl -sSL https://raw.githubusercontent.com/oar-cd/oar/main/install.sh | bash
```
