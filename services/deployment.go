package services

import "fmt"

type DeploymentStatus int

const (
	DeploymentStatusStarted DeploymentStatus = iota
	DeploymentStatusCompleted
	DeploymentStatusFailed
	DeploymentStatusUnknown
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
