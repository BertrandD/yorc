package events

import (
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"encoding/json"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
)

// A Subscriber is used to poll for new StatusEvents (StatusUpdate datas) and LogEvents
type Subscriber interface {
	StatusEvents(waitIndex uint64, timeout time.Duration) ([]StatusUpdate, uint64, error)
	LogsEvents(waitIndex uint64, timeout time.Duration) ([]json.RawMessage, uint64, error)
}

type consulPubSub struct {
	kv           *api.KV
	deploymentID string
}

// TODO : this should probably evolve...(temporary)
type consulSubscriber struct {
	kv *api.KV
}

// NewSubscriber returns an instance of Subscriber for a deployment
func NewSubscriber(kv *api.KV, deploymentID string) Subscriber {
	return &consulPubSub{kv: kv, deploymentID: deploymentID}
}

// InstanceStatusChange publishes a status change for a given instance of a given node
//
// InstanceStatusChange returns the published event id
func InstanceStatusChange(kv *api.KV, deploymentID, nodeName, instance, status string) (string, error) {
	id, err := storeStatusUpdateEvent(kv, deploymentID, InstanceStatusChangeType, nodeName+"\n"+status+"\n"+instance)
	if err != nil {
		return "", err
	}
	//TODO add log Optional fields
	SimpleLogEntry(INFO, deploymentID).RegisterAsString(fmt.Sprintf("Status for node %q, instance %q changed to %q", nodeName, instance, status))
	return id, nil
}

// DeploymentStatusChange publishes a status change for a given deployment
//
// DeploymentStatusChange returns the published event id
func DeploymentStatusChange(kv *api.KV, deploymentID, status string) (string, error) {
	id, err := storeStatusUpdateEvent(kv, deploymentID, DeploymentStatusChangeType, status)
	if err != nil {
		return "", err
	}
	//TODO add log Optional fields
	SimpleLogEntry(INFO, deploymentID).RegisterAsString(fmt.Sprintf("Status for deployment %q changed to %q", deploymentID, status))
	return id, nil
}

// CustomCommandStatusChange publishes a status change for a custom command
//
// CustomCommandStatusChange returns the published event id
func CustomCommandStatusChange(kv *api.KV, deploymentID, taskID, status string) (string, error) {
	id, err := storeStatusUpdateEvent(kv, deploymentID, CustomCommandStatusChangeType, taskID+"\n"+status)
	if err != nil {
		return "", err
	}
	//TODO add log Optional fields
	SimpleLogEntry(INFO, deploymentID).RegisterAsString(fmt.Sprintf("Status for custom-command %q changed to %q", taskID, status))
	return id, nil
}

// ScalingStatusChange publishes a status change for a scaling task
//
// ScalingStatusChange returns the published event id
func ScalingStatusChange(kv *api.KV, deploymentID, taskID, status string) (string, error) {
	id, err := storeStatusUpdateEvent(kv, deploymentID, ScalingStatusChangeType, taskID+"\n"+status)
	if err != nil {
		return "", err
	}
	//TODO add log Optional fields
	SimpleLogEntry(INFO, deploymentID).RegisterAsString(fmt.Sprintf("Status for scaling task %q changed to %q", taskID, status))
	return id, nil
}

// WorkflowStatusChange publishes a status change for a workflow task
//
// WorkflowStatusChange returns the published event id
func WorkflowStatusChange(kv *api.KV, deploymentID, taskID, status string) (string, error) {
	id, err := storeStatusUpdateEvent(kv, deploymentID, WorkflowStatusChangeType, taskID+"\n"+status)
	if err != nil {
		return "", err
	}
	//TODO add log Optional fields
	SimpleLogEntry(INFO, deploymentID).RegisterAsString(fmt.Sprintf("Status for workflow task %q changed to %q", taskID, status))
	return id, nil
}

// Create a KVPair corresponding to an event and put it to Consul under the event prefix,
// in a sub-tree corresponding to its deployment
// The eventType goes to the KVPair's Flags field
func storeStatusUpdateEvent(kv *api.KV, deploymentID string, eventType StatusUpdateType, data string) (string, error) {
	now := time.Now().Format(time.RFC3339Nano)
	eventsPrefix := path.Join(consulutil.EventsPrefix, deploymentID)
	p := &api.KVPair{Key: path.Join(eventsPrefix, now), Value: []byte(data), Flags: uint64(eventType)}
	_, err := kv.Put(p, nil)
	if err != nil {
		return "", err
	}
	return now, nil
}

func (cp *consulPubSub) StatusEvents(waitIndex uint64, timeout time.Duration) ([]StatusUpdate, uint64, error) {

	eventsPrefix := path.Join(consulutil.EventsPrefix, cp.deploymentID)
	// Get all the events for a deployment using a QueryOptions with the given waitIndex and with WaitTime corresponding to the given timeout
	kvps, qm, err := cp.kv.List(eventsPrefix, &api.QueryOptions{WaitIndex: waitIndex, WaitTime: timeout})
	events := make([]StatusUpdate, 0)

	if err != nil || qm == nil {
		return events, 0, err
	}

	log.Debugf("Found %d events before filtering, last index is %q", len(kvps), strconv.FormatUint(qm.LastIndex, 10))

	for _, kvp := range kvps {
		if kvp.ModifyIndex <= waitIndex {
			continue
		}

		eventTimestamp := strings.TrimPrefix(kvp.Key, eventsPrefix+"/")
		values := strings.Split(string(kvp.Value), "\n")
		eventType := StatusUpdateType(kvp.Flags)
		switch eventType {
		case InstanceStatusChangeType:
			if len(values) != 3 {
				return events, qm.LastIndex, errors.Errorf("Unexpected event value %q for event %q", string(kvp.Value), kvp.Key)
			}
			events = append(events, StatusUpdate{Timestamp: eventTimestamp, Type: eventType.String(), Node: values[0], Status: values[1], Instance: values[2]})
		case DeploymentStatusChangeType:
			if len(values) != 1 {
				return events, qm.LastIndex, errors.Errorf("Unexpected event value %q for event %q", string(kvp.Value), kvp.Key)
			}
			events = append(events, StatusUpdate{Timestamp: eventTimestamp, Type: eventType.String(), Status: values[0]})
		case CustomCommandStatusChangeType, ScalingStatusChangeType, WorkflowStatusChangeType:
			if len(values) != 2 {
				return events, qm.LastIndex, errors.Errorf("Unexpected event value %q for event %q", string(kvp.Value), kvp.Key)
			}
			events = append(events, StatusUpdate{Timestamp: eventTimestamp, Type: eventType.String(), TaskID: values[0], Status: values[1]})
		default:
			return events, qm.LastIndex, errors.Errorf("Unsupported event type %d for event %q", kvp.Flags, kvp.Key)
		}

	}

	log.Debugf("Found %d events after filtering", len(events))
	return events, qm.LastIndex, nil
}

// LogsEvents allows to return logs from Consul KV storage
func (cp *consulPubSub) LogsEvents(waitIndex uint64, timeout time.Duration) ([]json.RawMessage, uint64, error) {
	logs := make([]json.RawMessage, 0)

	eventsPrefix := path.Join(consulutil.LogsPrefix, cp.deploymentID)
	kvps, qm, err := cp.kv.List(eventsPrefix, &api.QueryOptions{WaitIndex: waitIndex, WaitTime: timeout})
	if err != nil || qm == nil {
		return logs, 0, err
	}
	log.Debugf("Found %d events before accessing index[%q]", len(kvps), strconv.FormatUint(qm.LastIndex, 10))
	for _, kvp := range kvps {
		if kvp.ModifyIndex <= waitIndex {
			continue
		}

		logs = append(logs, kvp.Value)
	}

	log.Debugf("Found %d events after index", len(logs))
	return logs, qm.LastIndex, nil
}

// GetStatusEventsIndex returns the latest index of InstanceStatus events for a given deployment
func GetStatusEventsIndex(kv *api.KV, deploymentID string) (uint64, error) {
	_, qm, err := kv.Get(path.Join(consulutil.EventsPrefix, deploymentID), nil)
	if err != nil {
		return 0, err
	}
	if qm == nil {
		return 0, errors.New("Failed to retrieve last index for events")
	}
	return qm.LastIndex, nil
}

// GetLogsEventsIndex returns the latest index of LogEntry events for a given deployment
func GetLogsEventsIndex(kv *api.KV, deploymentID string) (uint64, error) {
	_, qm, err := kv.Get(path.Join(consulutil.LogsPrefix, deploymentID), nil)
	if err != nil {
		return 0, err
	}
	if qm == nil {
		return 0, errors.New("Failed to retrieve last index for logs")
	}
	return qm.LastIndex, nil
}
