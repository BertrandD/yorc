package slurm

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
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
	chResult := make(chan string, 1)
	chOut := make(chan bool, 1)
	chErr := make(chan error)

	go parseSallocResponse(strings.NewReader(str), chResult, chOut, chErr)
	select {
	case <-chResult:
		require.Fail(t, "No jobID expected")
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
	chResult := make(chan string, 1)
	chOut := make(chan bool, 1)
	chErr := make(chan error)

	go parseSallocResponse(strings.NewReader(str), chResult, chOut, chErr)
	select {
	case jobID := <-chResult:
		require.Equal(t, "1881", jobID)
		return
	case err := <-chErr:
		require.Fail(t, "unexpected error", err.Error())
		return
	}

	require.Fail(t, "No response received")
}

// We test parsing the stdout line: "salloc: Granted job allocation 1881"
func TestParseSallocResponseWithExpectedGranted(t *testing.T) {
	str := "salloc: Granted job allocation 1881\n"
	chResult := make(chan string, 1)
	chOut := make(chan bool, 1)
	chErr := make(chan error)

	go parseSallocResponse(strings.NewReader(str), chResult, chOut, chErr)
	select {
	case jobID := <-chResult:
		fmt.Println("jobID = " + jobID)
		require.Equal(t, "1881", jobID)
		return
	case err := <-chErr:
		require.Fail(t, "unexpected error", err.Error())
		return
	}

	require.Fail(t, "No response received")
}

// We test parsing the stderr lines:
// "salloc: Job allocation 1882 has been revoked."
// "salloc: error: CPU count per node can not be satisfied"
// "salloc: error: Job submit/allocate failed: Requested node configuration is not available"
func TestParseSallocResponseWithExpectedRevokedAllocation(t *testing.T) {
	str := "salloc: Job allocation 1882 has been revoked.\nsalloc: error: CPU count per node can not be satisfied\nsalloc: error: Job submit/allocate failed: Requested node configuration is not available"
	chResult := make(chan string, 1)
	chOut := make(chan bool, 1)
	chErr := make(chan error)

	go parseSallocResponse(strings.NewReader(str), chResult, chOut, chErr)
	select {
	case jobID := <-chResult:
		require.Equal(t, "1881", jobID)
		return
	case err := <-chErr:
		require.Error(t, err)
		return
	}

	require.Fail(t, "No response received")
}
