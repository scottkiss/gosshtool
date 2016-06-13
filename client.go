package gosshtool

import (
	"bytes"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

type SSHClient struct {
	SSHClientConfig
	remoteConn  *ssh.Client
	isConnected bool
}

func (c *SSHClient) Connect() (conn *ssh.Client, err error) {
	if c.remoteConn != nil {
		return
	}
	port := "22"
	host := c.Host
	hstr := strings.SplitN(host, ":", 2)
	if len(hstr) == 2 {
		host = hstr[0]
		port = hstr[1]
	}

	config := makeConfig(c.User, c.Password, c.Privatekey)

	if c.DialTimeoutSecond > 0 {
		connNet, err := net.DialTimeout("tcp", host+":"+port, time.Duration(c.DialTimeoutSecond)*time.Second)
		if err != nil {
			return nil, err
		}
		sc, chans, reqs, err := ssh.NewClientConn(connNet, host+":"+port, config)
		if err != nil {
			return nil, err
		}
		conn = ssh.NewClient(sc, chans, reqs)
	} else {
		conn, err = ssh.Dial("tcp", host+":"+port, config)
		if err != nil {
			return
		}
	}
	log.Println("dial ssh success")
	c.remoteConn = conn
	return
}

func (c *SSHClient) Cmd(cmd string, sn *SshSession, deadline *time.Time) (output, errput string, currentSession *SshSession, err error) {
	if c.isConnected == false {
		_, err = c.Connect()
		if err != nil {
			return
		}
	}
	if sn == nil {
		currentSession, err = NewSession(c.remoteConn, deadline)
	} else {
		currentSession = sn
		currentSession.SetDeadline(deadline)
	}
	if err != nil {
		return
	}
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	currentSession.Stdout = &stdoutBuf
	currentSession.Stderr = &stderrBuf
	err = currentSession.Run(cmd)
	defer currentSession.Close()
	output = stdoutBuf.String()
	errput = stderrBuf.String()
	return
}

func (c *SSHClient) Pipe(rw ReadWriteCloser, pty *PtyInfo, deadline *time.Time) (currentSession *SshSession, err error) {
	if c.isConnected == false {
		_, err := c.Connect()
		if err != nil {
			return nil, err
		}
	}
	currentSession, err = NewSession(c.remoteConn, deadline)
	if err != nil {
		return
	}

	if err = currentSession.RequestPty(pty.Term, pty.H, pty.W, pty.Modes); err != nil {
		return
	}
	wc, err := currentSession.StdinPipe()
	if err != nil {
		return
	}
	go copyIO(wc, rw)

	r, err := currentSession.StdoutPipe()
	if err != nil {
		return
	}
	go copyIO(rw, r)
	er, err := currentSession.StderrPipe()
	if err != nil {
		return
	}
	go copyIO(rw, er)
	err = currentSession.Shell()
	if err != nil {
		return
	}
	err = currentSession.Wait()
	if err != nil {
		return
	}
	defer currentSession.Close()
	return
}

func copyIO(dst io.Writer, src io.Reader) (written int64, err error) {
	return io.Copy(dst, src)
}
