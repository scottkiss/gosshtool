package gosshtool

import (
	"bytes"
	"golang.org/x/crypto/ssh"
	"strings"
	"sync"
)

type SSHClient struct {
	SSHClientConfig
	remoteConns      map[string]*ssh.Client
	remoteConnsMutex sync.Mutex
}

func (c *SSHClient) getConnectionByHost(hostname string) (conn *ssh.Client, err error) {
	if c.remoteConns == nil {
		c.remoteConns = make(map[string]*ssh.Client)
	}
	c.remoteConnsMutex.Lock()
	conn = c.remoteConns[hostname]
	if conn != nil {
		return
	}
	c.remoteConnsMutex.Unlock()

	conn, err = c.getConnection()
	if err != nil {
		return
	}
	c.remoteConnsMutex.Lock()
	c.remoteConns[hostname] = conn
	c.remoteConnsMutex.Unlock()
	return
}

func (c *SSHClient) getConnection() (conn *ssh.Client, err error) {
	port := "22"
	host := c.Host
	hstr := strings.SplitN(host, ":", 2)
	if len(hstr) == 2 {
		host = hstr[0]
		port = hstr[1]
	}

	config := makeConfig(c.User, c.Password, c.Privatekey)
	conn, err = ssh.Dial("tcp", host+":"+port, config)
	return
}

func (c *SSHClient) Cmd(cmd string) (output, errput string, err error) {
	conn, err := c.getConnection()
	if err != nil {
		return
	}

	session, err := conn.NewSession()
	if err != nil {
		return
	}
	defer session.Close()
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf
	err = session.Run(cmd)
	output = stdoutBuf.String()
	errput = stderrBuf.String()
	return
}
