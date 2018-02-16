package tasks

import (
	"fmt"
	"path"
	"strconv"

	"strings"

	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/events"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
)

type taskDataNotFound struct {
	name   string
	taskID string
}

func (t taskDataNotFound) Error() string {
	return fmt.Sprintf("Data %q not found for task %q", t.name, t.taskID)
}

// IsTaskDataNotFoundError checks if an error is a task data not found error
func IsTaskDataNotFoundError(err error) bool {
	cause := errors.Cause(err)
	_, ok := cause.(taskDataNotFound)
	return ok
}

// GetTasksIdsForTarget returns IDs of tasks related to a given targetID
func GetTasksIdsForTarget(kv *api.KV, targetID string) ([]string, error) {
	tasksKeys, _, err := kv.Keys(consulutil.TasksPrefix+"/", "/", nil)
	if err != nil {
		return nil, err
	}
	tasks := make([]string, 0)
	for _, taskKey := range tasksKeys {
		kvp, _, err := kv.Get(path.Join(taskKey, "targetId"), nil)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp != nil && len(kvp.Value) > 0 && string(kvp.Value) == targetID {
			tasks = append(tasks, path.Base(taskKey))
		}
	}
	return tasks, nil
}

// GetTaskResultSet retrieves the task related resultSet in json string format
//
// If no resultSet is found, nil is returned instead
func GetTaskResultSet(kv *api.KV, taskID string) (string, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "resultSet"), nil)
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp != nil {
		return string(kvp.Value), nil
	}
	return "", nil
}

// GetTaskStatus retrieves the TaskStatus of a task
func GetTaskStatus(kv *api.KV, taskID string) (TaskStatus, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "status"), nil)
	if err != nil {
		return FAILED, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return FAILED, errors.Errorf("Missing status for task with id %q", taskID)
	}
	statusInt, err := strconv.Atoi(string(kvp.Value))
	if err != nil {
		return FAILED, errors.Wrapf(err, "Invalid task status:")
	}
	if statusInt < 0 || statusInt > int(CANCELED) {
		return FAILED, errors.Errorf("Invalid status for task with id %q: %q", taskID, string(kvp.Value))
	}
	return TaskStatus(statusInt), nil
}

// GetTaskType retrieves the TaskType of a task
func GetTaskType(kv *api.KV, taskID string) (TaskType, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "type"), nil)
	if err != nil {
		return Deploy, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return Deploy, errors.Errorf("Missing status for type with id %q", taskID)
	}
	typeInt, err := strconv.Atoi(string(kvp.Value))
	if err != nil {
		return Deploy, errors.Wrapf(err, "Invalid task type:")
	}
	if typeInt < 0 || typeInt > int(Query) {
		return Deploy, errors.Errorf("Invalid status for task with id %q: %q", taskID, string(kvp.Value))
	}
	return TaskType(typeInt), nil
}

// GetTaskTarget retrieves the targetID of a task
func GetTaskTarget(kv *api.KV, taskID string) (string, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "targetId"), nil)
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return "", errors.Errorf("Missing targetId for task with id %q", taskID)
	}
	return string(kvp.Value), nil
}

// GetTaskCreationDate retrieves the creationDate of a task
func GetTaskCreationDate(kv *api.KV, taskID string) (time.Time, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "creationDate"), nil)
	creationDate := time.Time{}
	if err != nil {
		return creationDate, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return creationDate, errors.Errorf("Missing creationDate for task with id %q", taskID)
	}
	err = creationDate.UnmarshalBinary(kvp.Value)
	if err != nil {
		return creationDate, errors.Wrapf(err, "Failed to get task creationDate for task with id %q", taskID)
	}
	return creationDate, nil
}

// TaskExists checks if a task with the given taskID exists
func TaskExists(kv *api.KV, taskID string) (bool, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "targetId"), nil)
	if err != nil {
		return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return false, nil
	}
	return true, nil
}

// CancelTask marks a task as Canceled
func CancelTask(kv *api.KV, taskID string) error {
	kvp := &api.KVPair{Key: path.Join(consulutil.TasksPrefix, taskID, ".canceledFlag"), Value: []byte("true")}
	_, err := kv.Put(kvp, nil)
	return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
}

// ResumeTask marks a task as Initial to allow it being resumed
func ResumeTask(kv *api.KV, taskID string) error {
	kvp := &api.KVPair{Key: path.Join(consulutil.TasksPrefix, taskID, "status"), Value: []byte(strconv.Itoa(int(INITIAL)))}
	_, err := kv.Put(kvp, nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	return nil
}

// DeleteTask allows to delete a stored task
func DeleteTask(kv *api.KV, taskID string) error {
	_, err := kv.DeleteTree(path.Join(consulutil.TasksPrefix, taskID), nil)
	return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
}

// TargetHasLivingTasks checks if a targetID has associated tasks in status INITIAL or RUNNING and returns the id and status of the first one found
func TargetHasLivingTasks(kv *api.KV, targetID string) (bool, string, string, error) {
	tasksKeys, _, err := kv.Keys(consulutil.TasksPrefix+"/", "/", nil)
	if err != nil {
		return false, "", "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, taskKey := range tasksKeys {
		kvp, _, err := kv.Get(path.Join(taskKey, "targetId"), nil)
		if err != nil {
			return false, "", "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp != nil && len(kvp.Value) > 0 && string(kvp.Value) == targetID {
			kvp, _, err := kv.Get(path.Join(taskKey, "status"), nil)
			taskID := path.Base(taskKey)
			if err != nil {
				return false, "", "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
			}
			if kvp == nil || len(kvp.Value) == 0 {
				return false, "", "", errors.Errorf("Missing status for task with id %q", taskID)
			}
			statusInt, err := strconv.Atoi(string(kvp.Value))
			if err != nil {
				return false, "", "", errors.Wrap(err, "Invalid task status")
			}
			switch TaskStatus(statusInt) {
			case INITIAL, RUNNING:
				return true, taskID, TaskStatus(statusInt).String(), nil
			}
		}
	}
	return false, "", "", nil
}

// GetTaskInput retrieves inputs for tasks
func GetTaskInput(kv *api.KV, taskID, inputName string) (string, error) {
	return GetTaskData(kv, taskID, path.Join("inputs", inputName))
}

// GetTaskData retrieves data for tasks
func GetTaskData(kv *api.KV, taskID, dataName string) (string, error) {
	kvP, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, dataName), nil)
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvP == nil {
		return "", errors.WithStack(taskDataNotFound{name: dataName, taskID: taskID})
	}
	return string(kvP.Value), nil
}

// GetInstances retrieve instances in the context of this task.
//
// Basically it checks if a list of instances is defined for this task for example in case of scaling.
// If not found it will returns the result of deployments.GetNodeInstancesIds(kv, deploymentID, nodeName).
func GetInstances(kv *api.KV, taskID, deploymentID, nodeName string) ([]string, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "nodes", nodeName), nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return deployments.GetNodeInstancesIds(kv, deploymentID, nodeName)
	}
	return strings.Split(string(kvp.Value), ","), nil
}

// GetTaskRelatedNodes returns the list of nodes that are specifically targeted by this task
//
// Currently it only appens for scaling tasks
func GetTaskRelatedNodes(kv *api.KV, taskID string) ([]string, error) {
	nodes, _, err := kv.Keys(path.Join(consulutil.TasksPrefix, taskID, "nodes")+"/", "/", nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for i := range nodes {
		nodes[i] = path.Base(nodes[i])
	}
	return nodes, nil
}

// IsTaskRelatedNode checks if the given nodeName is declared as a task related node
func IsTaskRelatedNode(kv *api.KV, taskID, nodeName string) (bool, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "nodes", nodeName), nil)
	if err != nil {
		return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	return kvp != nil, nil
}

// EmitTaskEvent emits a task event based on task type
func EmitTaskEvent(kv *api.KV, deploymentID, taskID string, taskType TaskType, status string) (eventID string, err error) {
	switch taskType {
	case CustomCommand:
		eventID, err = events.CustomCommandStatusChange(kv, deploymentID, taskID, strings.ToLower(status))
	case CustomWorkflow:
		eventID, err = events.WorkflowStatusChange(kv, deploymentID, taskID, strings.ToLower(status))
	case ScaleDown, ScaleUp:
		eventID, err = events.ScalingStatusChange(kv, deploymentID, taskID, strings.ToLower(status))
	}
	return
}

// GetTaskRelatedSteps returns the steps of the related workflow
func GetTaskRelatedSteps(kv *api.KV, taskID string) ([]TaskStep, error) {
	steps := make([]TaskStep, 0)

	kvps, _, err := kv.List(path.Join(consulutil.WorkflowsPrefix, taskID), nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	for _, kvp := range kvps {
		steps = append(steps, TaskStep{Name: path.Base(kvp.Key), Status: string(kvp.Value)})
	}
	return steps, nil
}

// TaskStepExists checks if a task step exists with a stepID and related to a given taskID and returns it
func TaskStepExists(kv *api.KV, taskID, stepID string) (bool, *TaskStep, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.WorkflowsPrefix, taskID, stepID), nil)
	if err != nil {
		return false, nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return false, nil, nil
	}
	return true, &TaskStep{Name: stepID, Status: string(kvp.Value)}, nil
}

// UpdateTaskStepStatus allows to update the task step status
func UpdateTaskStepStatus(kv *api.KV, taskID string, step *TaskStep) error {
	kvp := &api.KVPair{Key: path.Join(consulutil.WorkflowsPrefix, taskID, step.Name), Value: []byte(step.Status)}
	_, err := kv.Put(kvp, nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	return nil
}

// CheckTaskStepStatusChange checks if a status change is allowed
func CheckTaskStepStatusChange(before, after string) (bool, error) {
	if before == after {
		return false, errors.New("Final and initial status are identical: nothing to do")
	}
	stBefore, err := ParseTaskStepStatus(before)
	if err != nil {
		return false, err
	}
	stAfter, err := ParseTaskStepStatus(after)
	if err != nil {
		return false, err
	}

	if stBefore != TaskStepStatusERROR || stAfter != TaskStepStatusDONE {
		return false, nil
	}
	return true, nil
}
