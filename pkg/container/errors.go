package container

import (
	"fmt"
	"strings"
)

// ContainerError represents an error that occurred in the container
type ContainerError struct {
	Code    string
	Message string
	Cause   error
}

// Error implements the error interface
func (e *ContainerError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the cause of the error
func (e *ContainerError) Unwrap() error {
	return e.Cause
}

// ComponentNotFoundError returns an error for when a component is not found
func ComponentNotFoundError(name string) *ContainerError {
	return &ContainerError{
		Code:    "COMPONENT_NOT_FOUND",
		Message: fmt.Sprintf("component with name '%s' not found", name),
	}
}

// ComponentAlreadyRegisteredError returns an error for when a component is already registered
func ComponentAlreadyRegisteredError(name string) *ContainerError {
	return &ContainerError{
		Code:    "COMPONENT_ALREADY_REGISTERED",
		Message: fmt.Sprintf("component with name '%s' already registered", name),
	}
}

// CircularDependencyError returns an error for when a circular dependency is detected
func CircularDependencyError(cycle []string) *ContainerError {
	return &ContainerError{
		Code:    "CIRCULAR_DEPENDENCY",
		Message: fmt.Sprintf("circular dependency detected: %s", strings.Join(cycle, " -> ")),
	}
}

// ComponentInitializationError returns an error for when a component fails to initialize
func ComponentInitializationError(name string, err error) *ContainerError {
	return &ContainerError{
		Code:    "COMPONENT_INITIALIZATION_FAILED",
		Message: fmt.Sprintf("component '%s' failed to initialize", name),
		Cause:   err,
	}
}

// ComponentTypeError returns an error for when a component has an unexpected type
func ComponentTypeError(name string, expected, actual string) *ContainerError {
	return &ContainerError{
		Code:    "COMPONENT_TYPE_ERROR",
		Message: fmt.Sprintf("component '%s' is not of expected type: expected %s, got %s", name, expected, actual),
	}
}

// ConfigurationError returns an error for when configuration is invalid
func ConfigurationError(msg string, cause error) *ContainerError {
	return &ContainerError{
		Code:    "CONFIGURATION_ERROR",
		Message: msg,
		Cause:   cause,
	}
}
