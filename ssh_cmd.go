package main

// inspired by https://gist.github.com/kiyor/7817632

import (
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"os"
)

type SshCmdRunner struct {
	Host string
	Cmd  string

	buf, errbuf io.Reader
	killed      bool

	sshConfig *ssh.ClientConfig
	client    *ssh.Client
	session   *ssh.Session
}

// Build a new struct that manages connecting / running cmd / killing
//
func NewSshCmdRunner(host, username, credentials string) (*SshCmdRunner, error) {
	return &SshCmdRunner{
		Host:   host,
		killed: false,
		sshConfig: &ssh.ClientConfig{
			User:            username,
			Auth:            buildAuth(credentials),
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		},
	}, nil
}

// Return if Kill() has been called on this
//
func (s *SshCmdRunner) Killed() bool {
	return s.killed
}

// Send a sigabort to the end processes, if we've connected
//
func (s *SshCmdRunner) Kill() {
	s.killed = true

	if s.session == nil {
		return
	}

	// Send sigabort, close everything and take cover
	s.session.Signal(ssh.SIGABRT)
	s.session.Close()
	s.client.Close()
}

func (s *SshCmdRunner) Clear() {
	// noop
}

// Return data from this commands stdout buffer
//
func (s *SshCmdRunner) Stdout() ([]byte, error) {
	if s.buf == nil {
		return []byte{}, nil
	}

	buff := make([]byte, 1024)

	read, err := s.buf.Read(buff)
	if err != nil {
		return nil, err
	}

	if read == 0 {
		return []byte{}, nil
	}

	return buff[:read], nil
}

// Return data from this commands stderr buffer
//
func (s *SshCmdRunner) Stderr() ([]byte, error) {
	if s.buf == nil {
		return []byte{}, nil
	}

	buff := make([]byte, 1024)

	read, err := s.errbuf.Read(buff)
	if err != nil {
		return nil, err
	}

	if read == 0 {
		return []byte{}, nil
	}

	return buff[:read], nil
}

// Execute the given command.
//  This handles connecting (dial), auth & finally executing the command.
//
func (s *SshCmdRunner) Execute(cmd string) (err error) {
	// Connect using the current config and run the given cmd

	s.Cmd = cmd
	client, err := ssh.Dial("tcp", s.Host, s.sshConfig)
	if err != nil {
		return err
	}
	s.client = client
	defer s.client.Close()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	s.session = session
	defer s.session.Close()

	errPipe, err := s.session.StderrPipe()
	if err != nil {
		return err
	}

	outPipe, err := s.session.StdoutPipe()
	if err != nil {
		return err
	}

	s.buf = outPipe
	s.errbuf = errPipe

	if err = session.Run(cmd); err != nil {
		return err
	}

	return err
}

// Builds an ssh.AuthMethod from a string.
//  Assumes it's a ssh public key, then fallsback to password.
//
func buildAuth(key string) []ssh.AuthMethod {
	m := []ssh.AuthMethod{}
	k, err := loadKey(key)

	if err == nil {
		m = append(m, ssh.PublicKeys(k))
	} else {
		m = append(m, ssh.Password(key))
	}

	return m
}

// Load ssh public key give it's filepath.
//
func loadKey(fpath string) (key ssh.Signer, err error) {
	_, err = os.Stat(fpath)
	if err != nil {
		return key, err
	}

	buf, err := ioutil.ReadFile(fpath)
	if err != nil {
		return key, err
	}

	key, err = ssh.ParsePrivateKey(buf)
	if err != nil {
		return key, err
	}
	return key, nil
}
