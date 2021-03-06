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

package workflow

import (
	"path"
	"strconv"

	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/tasks"
)

type task struct {
	ID           string
	TargetID     string
	status       tasks.TaskStatus
	TaskType     tasks.TaskType
	creationDate time.Time
	taskLock     *api.Lock
	kv           *api.KV
}

func (t *task) releaseLock() {
	t.taskLock.Unlock()
	t.taskLock.Destroy()
}

func (t *task) Status() tasks.TaskStatus {
	return t.status
}

func (t *task) WithStatus(status tasks.TaskStatus) error {
	p := &api.KVPair{Key: path.Join(consulutil.TasksPrefix, t.ID, "status"), Value: []byte(strconv.Itoa(int(status)))}
	_, err := t.kv.Put(p, nil)
	t.status = status
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	_, err = tasks.EmitTaskEvent(t.kv, t.TargetID, t.ID, t.TaskType, t.status.String())

	return err
}
