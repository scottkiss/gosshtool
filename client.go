package gosshtool

import (
	"bytes"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	DEFAULT_CHUNK_SIZE        = 65536
	MIN_CHUNKS                = 10
	THROUGHPUT_SLEEP_INTERVAL = 100
	MIN_THROUGHPUT            = DEFAULT_CHUNK_SIZE * MIN_CHUNKS * (1000 / THROUGHPUT_SLEEP_INTERVAL)
)

var (
	maxThroughputChan  = make(chan bool, MIN_CHUNKS)
	maxThroughput      uint64
	maxThroughputMutex sync.Mutex
)

type SSHClient struct {
	SSHClientConfig
	remoteConn  *ssh.Client
	isConnected bool
}

func (c *SSHClient) maxThroughputControl() {
	for {
		if c.MaxDataThroughput > 0 && c.MaxDataThroughput < MIN_THROUGHPUT {
			log.Panicf("Minimal throughput is %d Bps", MIN_THROUGHPUT)
		}
		maxThroughputMutex.Lock()
		throughput := c.MaxDataThroughput
		maxThroughputMutex.Unlock()
		chunks := throughput / DEFAULT_CHUNK_SIZE * THROUGHPUT_SLEEP_INTERVAL / 1000
		if chunks < MIN_CHUNKS {
			chunks = MIN_CHUNKS
		}
		for i := uint64(0); i < chunks; i++ {
			maxThroughputChan <- true
		}
		if throughput > 0 {
			time.Sleep(THROUGHPUT_SLEEP_INTERVAL * time.Millisecond)
		}
	}
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

func (c *SSHClient) TransferData(target string, data []byte) (stdout, stderr string, err error) {
	go c.maxThroughputControl()

	if c.isConnected == false {
		_, err = c.Connect()
		if err != nil {
			return
		}
	}
	currentSession, err := NewSession(c.remoteConn, nil, 0)
	if err != nil {
		return
	}
	defer currentSession.Close()
	cmd := "cat >'" + strings.Replace(target, "'", "'\\''", -1) + "'"
	stdinPipe, err := currentSession.StdinPipe()
	if err != nil {
		return
	}
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	currentSession.Stdout = &stdoutBuf
	currentSession.Stderr = &stderrBuf
	err = currentSession.session.Start(cmd)
	if err != nil {
		return
	}
	for start, max := 0, len(data); start < max; start += DEFAULT_CHUNK_SIZE {
		<-maxThroughputChan
		end := start + DEFAULT_CHUNK_SIZE
		if end > max {
			end = max
		}
		_, err = stdinPipe.Write(data[start:end])
		if err != nil {
			return
		}
	}
	err = stdinPipe.Close()
	if err != nil {
		return
	}
	err = currentSession.Wait()
	stdout = stdoutBuf.String()
	stderr = stderrBuf.String()
	return
}

func (c *SSHClient) Cmd(cmd string, sn *SshSession, deadline *time.Time, idleTimeout int) (output, errput string, currentSession *SshSession, err error) {
	if c.isConnected == false {
		_, err = c.Connect()
		if err != nil {
			return
		}
	}
	if sn == nil {
		currentSession, err = NewSession(c.remoteConn, deadline, idleTimeout)
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

func (c *SSHClient) Pipe(rw ReadWriteCloser, pty *PtyInfo, deadline *time.Time, idleTimeout int) (currentSession *SshSession, err error) {
	if c.isConnected == false {
		_, err := c.Connect()
		if err != nil {
			return nil, err
		}
	}
	currentSession, err = NewSession(c.remoteConn, deadline, idleTimeout)
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

	go CopyIOAndUpdateSessionDeadline(wc, rw, currentSession)

	r, err := currentSession.StdoutPipe()
	if err != nil {
		return
	}
	go CopyIOAndUpdateSessionDeadline(rw, r, currentSession)
	er, err := currentSession.StderrPipe()
	if err != nil {
		return
	}
	go CopyIOAndUpdateSessionDeadline(rw, er, currentSession)
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
