package domain

import "fmt"

// ProjectStatus represents the runtime status of a project
type ProjectStatus int

const (
	ProjectStatusUnknown ProjectStatus = iota
	ProjectStatusRunning
	ProjectStatusStopped
	ProjectStatusError
)

func (s ProjectStatus) String() string {
	switch s {
	case ProjectStatusRunning:
		return "running"
	case ProjectStatusStopped:
		return "stopped"
	case ProjectStatusError:
		return "error"
	case ProjectStatusUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}

func ParseProjectStatus(s string) (ProjectStatus, error) {
	switch s {
	case "running":
		return ProjectStatusRunning, nil
	case "stopped":
		return ProjectStatusStopped, nil
	case "error":
		return ProjectStatusError, nil
	case "unknown":
		return ProjectStatusUnknown, nil
	default:
		return ProjectStatusUnknown, fmt.Errorf("invalid project status: %q", s)
	}
}

// DeploymentStatus represents the status of a deployment
type DeploymentStatus int

const (
	DeploymentStatusUnknown DeploymentStatus = iota
	DeploymentStatusStarted
	DeploymentStatusCompleted
	DeploymentStatusFailed
)

func (s DeploymentStatus) String() string {
	switch s {
	case DeploymentStatusStarted:
		return "started"
	case DeploymentStatusCompleted:
		return "completed"
	case DeploymentStatusFailed:
		return "failed"
	case DeploymentStatusUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}

func ParseDeploymentStatus(s string) (DeploymentStatus, error) {
	switch s {
	case "started":
		return DeploymentStatusStarted, nil
	case "completed":
		return DeploymentStatusCompleted, nil
	case "failed":
		return DeploymentStatusFailed, nil
	case "unknown":
		return DeploymentStatusUnknown, nil
	default:
		return DeploymentStatusUnknown, fmt.Errorf("invalid deployment status: %q", s)
	}
}
