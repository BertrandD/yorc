package ansible

import (
	"context"
	"math/rand"
	"time"

	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov"
)

type defaultExecutor struct {
	r *rand.Rand
}

// NewExecutor returns an Executor
func NewExecutor() prov.OperationExecutor {
	return &defaultExecutor{r: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

func (e *defaultExecutor) ExecOperation(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation) error {
	consulClient, err := conf.GetConsulClient()
	if err != nil {
		return err
	}
	kv := consulClient.KV()
	exec, err := newExecution(kv, conf, taskID, deploymentID, nodeName, operation)
	if err != nil {
		if IsOperationNotImplemented(err) {
			log.Debugf("Voluntary bypassing error: %s.", err.Error())
			return nil
		}
		return err
	}

	// Execute operation
	err = exec.execute(ctx, conf.AnsibleConnectionRetries != 0)
	if err == nil {
		return nil
	}
	if !IsRetriable(err) {
		return err
	}

	// Retry operation if error is retriable and AnsibleConnectionRetries > 0
	log.Debugf("Ansible Connection Retries:%d", conf.AnsibleConnectionRetries)
	if conf.AnsibleConnectionRetries > 0 {
		for i := 0; i < conf.AnsibleConnectionRetries; i++ {
			log.Printf("Deployment: %q, Node: %q, Operation: %s: Caught a retriable error from Ansible: '%s'. Let's retry in few seconds (%d/%d)", deploymentID, nodeName, operation, err, i+1, conf.AnsibleConnectionRetries)
			time.Sleep(time.Duration(e.r.Int63n(10)) * time.Second)
			err = exec.execute(ctx, i != 0)
			if err == nil {
				return nil
			}
			if !IsRetriable(err) {
				return err
			}
		}

		log.Printf("Deployment: %q, Node: %q, Operation: %s: Giving up retries for Ansible error: '%s' (%d/%d)", deploymentID, nodeName, operation, err, conf.AnsibleConnectionRetries, conf.AnsibleConnectionRetries)
	}

	return err
}
