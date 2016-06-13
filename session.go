package gosshtool

import (
	"bytes"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"time"
)

type SshSession struct {
	id       string
	session  *ssh.Session
	Stdout   *bytes.Buffer
	Stderr   *bytes.Buffer
	deadline *time.Time
}

func (sc *SshSession) Run(cmd string) (err error) {
	sc.session.Stdout = sc.Stdout
	sc.session.Stderr = sc.Stderr
	return sc.session.Run(cmd)
}

func (sc *SshSession) RequestPty(term string, h int, w int, termmodes ssh.TerminalModes) (err error) {
	return sc.session.RequestPty(term, h, w, termmodes)
}

func (sc *SshSession) StdinPipe() (io.WriteCloser, error) {
	return sc.session.StdinPipe()
}

func (sc *SshSession) StdoutPipe() (io.Reader, error) {
	return sc.session.StdoutPipe()
}

func (sc *SshSession) StderrPipe() (io.Reader, error) {
	return sc.session.StderrPipe()
}

func (sc *SshSession) SetDeadline(deadline *time.Time) {
	sc.deadline = deadline
}

func (sc *SshSession) Shell() error {
	return sc.session.Shell()
}

func (sc *SshSession) Wait() error {
	return sc.session.Wait()
}

func (sc *SshSession) Close() error {
	return sc.session.Close()
}

func NewSession(conn *ssh.Client, deadline *time.Time) (ss *SshSession, err error) {
	session, err := conn.NewSession()
	if err != nil {
		return nil, err
	}
	sshSession := new(SshSession)
	sshSession.session = session
	sshSession.deadline = deadline
	sshSession.id = Rand().Hex()
	//check session timeout
	go sshSession.checkSessionTimeout()
	return sshSession, nil
}

func (sc *SshSession) checkSessionTimeout() {
	timeout := make(chan bool, 1)
	go func() {
		t := time.NewTicker(time.Second * 1)
		for {
			<-t.C
			if !(*sc.deadline).IsZero() && time.Now().After(*sc.deadline) {
				timeout <- true
			}
		}
	}()
	ch := make(chan int)
	select {
	case <-ch:
	case <-timeout:
		log.Println("timeout!")
		sc.Close()
	}
}
