package gosshtool

import (
	"bytes"
	"golang.org/x/crypto/ssh"
	"log"
	"strings"
)

type SSHClient struct {
	Host       string
	User       string
	Password   string
	Privatekey string
}

func (c *SSHClient) getConnection() (conn *ssh.Client, err error) {
	port := 22
	host := c.Host
	hstr := strings.SplitN(host, ":", 2)
	if len(hstr) == 2 {
		host = hstr[0]
		port = hstr[1]
	}

	if c.Password == "" && c.User == "" {
		log.Fatal("No password or private key available")
	}
	config := &ssh.ClientConfig{
		User: c.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(c.Password),
		},
	}
	if c.Privatekey != "" {
		log.Println(c.Privatekey)
		signer, err := ssh.ParsePrivateKey([]byte(c.PrivateKey))
		if err != nil {
			log.Fatalf("ssh.ParsePrivateKey error:%v", err)
		}
		clientkey := ssh.PublicKeys(signer)
		config = &ssh.ClientConfig{
			User: c.User,
			Auth: []ssh.AuthMethod{
				clientkey,
			},
		}

	}

	conn, err := ssh.Dial("tcp", host+":"+port, config)
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
