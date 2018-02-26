// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tasks

import (
	"fmt"
)

//go:generate stringer -type=TaskStatus,TaskType -output=structs_string.go structs.go
//go:generate go-enum -f=structs.go --lower

// A TaskType determines the type of a Task
type TaskType int

const (
	// Deploy defines a Task of type "deploy"
	Deploy TaskType = iota
	// UnDeploy defines a Task of type "undeploy"
	UnDeploy
	// ScaleOut defines a Task of type "scale-out"
	ScaleOut
	// ScaleIn defines a Task of type "scale-in"
	ScaleIn
	// Purge defines a Task of type "purge"
	Purge
	// CustomCommand defines a Task of type "custom-command"
	CustomCommand
	// CustomWorkflow defines a Task of type "CustomWorkflow"
	CustomWorkflow
	// Query defines a Task of type "Query"
	Query
	// NOTE: if a new task type should be added then change validity check on GetTaskType
)

// TaskStatus represents the status of a Task
type TaskStatus int

// TaskStepStatus x ENUM(
// INITIAL,
// RUNNING,
// DONE,
// ERROR,
// CANCELED
// )
type TaskStepStatus int

const (
	// INITIAL is the initial status of a that haven't run yet
	INITIAL TaskStatus = iota
	// RUNNING is the status of a task that is currently processed
	RUNNING
	// DONE is the status of a task successful task
	DONE
	// FAILED is the status of a failed task
	FAILED
	// CANCELED is the status of a canceled task
	CANCELED
	// NOTE: if a new status should be added then change validity check on GetTaskStatus
)

// TaskStep represents a step related to the task
type TaskStep struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type anotherLivingTaskAlreadyExistsError struct {
	taskID   string
	targetID string
	status   string
}

func (e anotherLivingTaskAlreadyExistsError) Error() string {
	return fmt.Sprintf("Task with id %q and status %q already exists for target %q", e.taskID, e.status, e.targetID)
}

// IsAnotherLivingTaskAlreadyExistsError checks if an error is due to the fact that another task is currently running
// If true, it returns the taskID of the currently running task
func IsAnotherLivingTaskAlreadyExistsError(err error) (bool, string) {
	e, ok := err.(anotherLivingTaskAlreadyExistsError)
	if ok {
		return ok, e.taskID
	}
	return ok, ""
}
