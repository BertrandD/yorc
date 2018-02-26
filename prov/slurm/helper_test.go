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

package slurm

import (
	"errors"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

// MockSSHSession allows to mock an SSH session
type MockSSHClient struct {
	MockRunCommand func(string) (string, error)
}

// RunCommand to mock a command ran via SSH
func (s *MockSSHClient) RunCommand(cmd string) (string, error) {
	if s.MockRunCommand != nil {
		return s.MockRunCommand(cmd)
	}
	return "", nil
}

func TestGetAttribute(t *testing.T) {
	t.Parallel()
	s := &MockSSHClient{
		MockRunCommand: func(cmd string) (string, error) {
			return "CUDA_VISIBLE_DEVICES=NoDevFiles", nil
		},
	}
	value, err := getAttribute(s, "cuda_visible_devices", "1234", "myNodeName")
	require.Nil(t, err)
	require.Equal(t, "NoDevFiles", value)
}

func TestGetAttributeWithUnknownKey(t *testing.T) {
	t.Parallel()
	s := &MockSSHClient{}
	value, err := getAttribute(s, "unknown_key", "1234", "myNodeName")
	require.Equal(t, "", value)
	require.Error(t, err, "unknown key error expected")
}

func TestGetAttributeWithFailure(t *testing.T) {
	t.Parallel()
	s := &MockSSHClient{
		MockRunCommand: func(cmd string) (string, error) {
			return "", errors.New("expected failure")
		},
	}
	value, err := getAttribute(s, "unknown_key", "1234", "myNodeName")
	require.Equal(t, "", value)
	require.Error(t, err, "expected failure expected")
}

func TestGetAttributeWithMalformedStdout(t *testing.T) {
	s := &MockSSHClient{
		MockRunCommand: func(cmd string) (string, error) {
			return "MALFORMED_VALUE", nil
		},
	}
	value, err := getAttribute(s, "unknown_key", "1234", "myNodeName")
	require.Equal(t, "", value)
	require.Error(t, err, "expected property/value is malformed")
}

// We test parsing the stderr line: ""
func TestParseSallocResponseWithEmpty(t *testing.T) {
	str := ""
	chResult := make(chan allocationResponse)
	chErr := make(chan error)

	go parseSallocResponse(strings.NewReader(str), chResult, chErr)
	select {
	case <-chResult:
		require.Fail(t, "No response expected")
		return
	case err := <-chErr:
		require.Fail(t, "unexpected error", err.Error())
		return
	default:
		require.True(t, true)
	}
}

// We test parsing the stderr line: "salloc: Pending job allocation 1881"
func TestParseSallocResponseWithExpectedPending(t *testing.T) {
	str := "salloc: Pending job allocation 1881\n"
	chResult := make(chan allocationResponse)
	chErr := make(chan error)

	var res allocationResponse

	go parseSallocResponse(strings.NewReader(str), chResult, chErr)
	select {
	case res = <-chResult:
		require.Equal(t, "1881", res.jobID)
		require.Equal(t, false, res.granted)
		return
	case err := <-chErr:
		require.Fail(t, "unexpected error", err.Error())
		return
	case <-time.After(1 * time.Second):
		require.Fail(t, "No response received")
	}
}

//salloc: Required node not available (down, drained or reserved)
//salloc: Pending job allocation 2220
//salloc: job 2220 queued and waiting for resources
func TestParseSallocResponseWithExpectedPendingInOtherThanFirstLine(t *testing.T) {
	str := "salloc: Required node not available (down, drained or reserved)\nsalloc: Pending job allocation 2220\nsalloc: job 2220 queued and waiting for resources"
	chResult := make(chan allocationResponse)
	chErr := make(chan error)

	var res allocationResponse

	go parseSallocResponse(strings.NewReader(str), chResult, chErr)
	select {
	case res = <-chResult:
		require.Equal(t, "2220", res.jobID)
		require.Equal(t, false, res.granted)
		return
	case err := <-chErr:
		require.Fail(t, "unexpected error", err.Error())
		return
	case <-time.After(1 * time.Second):
		require.Fail(t, "No response received")
	}
}

// We test parsing the stdout line: "salloc: Granted job allocation 1881"
func TestParseSallocResponseWithExpectedGranted(t *testing.T) {
	str := "salloc: Granted job allocation 1881\n"
	chResult := make(chan allocationResponse)
	chErr := make(chan error)

	var res allocationResponse

	go parseSallocResponse(strings.NewReader(str), chResult, chErr)
	select {
	case res = <-chResult:
		require.Equal(t, "1881", res.jobID)
		require.Equal(t, true, res.granted)
		return
	case err := <-chErr:
		require.Fail(t, "unexpected error", err.Error())
		return
	case <-time.After(1 * time.Second):
		require.Fail(t, "No response received")
	}
}

// We test parsing the stderr lines:
// "salloc: Job allocation 1882 has been revoked."
// "salloc: error: CPU count per node can not be satisfied"
// "salloc: error: Job submit/allocate failed: Requested node configuration is not available"
func TestParseSallocResponseWithExpectedRevokedAllocation(t *testing.T) {
	str := "salloc: Job allocation 1882 has been revoked.\nsalloc: error: CPU count per node can not be satisfied\nsalloc: error: Job submit/allocate failed: Requested node configuration is not available"
	chResult := make(chan allocationResponse)
	chErr := make(chan error)

	go parseSallocResponse(strings.NewReader(str), chResult, chErr)
	select {
	case <-chResult:
		require.Fail(t, "No expected response")
		return
	case err := <-chErr:
		require.Error(t, err)
		return
	case <-time.After(1 * time.Second):
		require.Fail(t, "No response received")
	}
}
