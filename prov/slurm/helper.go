package slurm

import (
	"bufio"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"novaforge.bull.com/starlings-janus/janus/helper/sshutil"
	"regexp"
	"strings"
)

// getAttribute allows to return an attribute with defined key from specific treatment
func getAttribute(client sshutil.Client, key string, jobID, nodeName string) (string, error) {
	switch key {
	case "cuda_visible_devices":
		if jobID != "" {
			cmd := fmt.Sprintf("srun --jobid=%s env|grep CUDA_VISIBLE_DEVICES", jobID)
			stdout, err := client.RunCommand(cmd)
			if err != nil {
				return "", errors.Wrapf(err, "Unable to retrieve (%s) for node:%q", key, nodeName)
			}
			value, err := getEnvValue(stdout)
			if err != nil {
				return "", errors.Wrapf(err, "Unable to retrieve (%s) for node:%q", key, nodeName)
			}
			return value, nil
		}
		return "", nil
	default:
		return "", fmt.Errorf("unknown key:%s", key)
	}
}

// getEnvValue allows to return the value in a formatted string as "property=value"
func getEnvValue(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	if strings.ContainsRune(s, '=') {
		propVal := strings.Split(s, "=")
		if len(propVal) == 2 {
			return propVal[1], nil
		}
		return "", errors.New("property/value is malformed")
	}
	return "", errors.New("property/value is malformed")
}

// parseSallocResponse parses stderr and stdout for salloc command
// Below are classic examples:
// salloc: Granted job allocation 1881
// salloc: Pending job allocation 1881

//salloc: Job allocation 1882 has been revoked.
//salloc: error: CPU count per node can not be satisfied
//salloc: error: Job submit/allocate failed: Requested node configuration is not available
func parseSallocResponse(r io.Reader, chRes chan allocationResponse, chErr chan error) {
	var (
		jobID  string
		err    error
		strErr string
	)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		reGranted := regexp.MustCompile(reSallocGranted)
		rePending := regexp.MustCompile(reSallocPending)
		if reGranted.MatchString(line) {
			// expected line: "salloc: Granted job allocation 1881"
			if jobID, err = parseJobID(line, reGranted); err != nil {
				chErr <- err
				return
			}
			chRes <- allocationResponse{jobID: jobID, granted: true}
			return
		} else if rePending.MatchString(line) {
			// expected line: "salloc: Pending job allocation 1881"
			if jobID, err = parseJobID(line, rePending); err != nil {
				chErr <- err
				return
			}
			chRes <- allocationResponse{jobID: jobID, granted: false}
			return
		}
		// If no expected lines found, we retrieve the full lines
		if strErr != "" {
			strErr += " "
		}
		strErr += line
	}
	if err := scanner.Err(); err != nil {
		chErr <- errors.Wrap(err, "An error occurred scanning stdout/stderr")
		return
	}
	if len(strErr) > 0 {
		chErr <- errors.Errorf("salloc command returned an error:%q", strErr)
	}
	return
}

func parseJobID(str string, regexp *regexp.Regexp) (string, error) {
	subMatch := regexp.FindStringSubmatch(str)
	if subMatch != nil && len(subMatch) == 2 {
		return subMatch[1], nil
	}
	return "", errors.Errorf("Unable to parse std:%q for retrieving jobID", str)
}

func cancelJobID(jobID string, client *sshutil.SSHClient) error {
	scancelCmd := fmt.Sprintf("scancel %s", jobID)
	sCancelOutput, err := client.RunCommand(scancelCmd)
	if err != nil {
		return errors.Wrapf(err, "Failed to cancel Slurm job: %s:", sCancelOutput)
	}
	return nil
}
