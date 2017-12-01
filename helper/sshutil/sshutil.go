package sshutil

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"io"
	"novaforge.bull.com/starlings-janus/janus/log"
)

// Client is interface allowing running command
type Client interface {
	RunCommand(string) (string, error)
}

// SSHSessionWrapper is a wrapper with a piped SSH session
type SSHSessionWrapper struct {
	Session *ssh.Session
	Stdout  io.Reader
	Stderr  io.Reader
}

// SSHClient is a client SSH
type SSHClient struct {
	Config *ssh.ClientConfig
	Host   string
	Port   int
}

// GetSessionWrapper allows to return a session wrapper in order to handle stdout/stderr for running long synchronous commands
func (client *SSHClient) GetSessionWrapper() (*SSHSessionWrapper, error) {
	var ps = &SSHSessionWrapper{}
	var err error
	ps.Session, err = client.newSession()
	if err != nil {
		return nil, errors.Wrap(err, "Unable to prepare SSH command")
	}

	log.Debug("[SSHSession] Add Stderr/Stdout pipelines")
	ps.Stdout, err = ps.Session.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "Unable to setup stdout for session")
	}

	ps.Stderr, err = ps.Session.StderrPipe()
	if err != nil {
		return nil, errors.Wrap(err, "Unable to setup stderr for session")
	}

	return ps, nil
}

// RunCommand allows to run a specified command
func (client *SSHClient) RunCommand(cmd string) (string, error) {
	session, err := client.newSession()
	if err != nil {
		return "", errors.Wrap(err, "Unable to setup stdout for session")
	}
	defer session.Close()
	var b bytes.Buffer
	session.Stderr = &b
	session.Stdout = &b

	log.Debugf("[SSHSession] %q", cmd)
	err = session.Run(cmd)
	return b.String(), err
}

func (client *SSHClient) newSession() (*ssh.Session, error) {
	connection, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", client.Host, client.Port), client.Config)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to open SSH connection")
	}

	session, err := connection.NewSession()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create session")
	}

	return session, nil
}

// RunCommand allows to run a specified command from a session wrapper in order to handle stdout/stderr during long synchronous commands
func (sw *SSHSessionWrapper) RunCommand(cmd string) error {
	defer sw.Session.Close()
	log.Debugf("[SSHSession] %q", cmd)
	err := sw.Session.Run(cmd)
	return err
}
