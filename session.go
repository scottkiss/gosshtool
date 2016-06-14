package gosshtool

import (
	"bytes"
	"errors"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"time"
)

// POSIX terminal mode flags as listed in RFC 4254 Section 8.
const (
	tty_OP_END    = 0
	VINTR         = 1
	VQUIT         = 2
	VERASE        = 3
	VKILL         = 4
	VEOF          = 5
	VEOL          = 6
	VEOL2         = 7
	VSTART        = 8
	VSTOP         = 9
	VSUSP         = 10
	VDSUSP        = 11
	VREPRINT      = 12
	VWERASE       = 13
	VLNEXT        = 14
	VFLUSH        = 15
	VSWTCH        = 16
	VSTATUS       = 17
	VDISCARD      = 18
	IGNPAR        = 30
	PARMRK        = 31
	INPCK         = 32
	ISTRIP        = 33
	INLCR         = 34
	IGNCR         = 35
	ICRNL         = 36
	IUCLC         = 37
	IXON          = 38
	IXANY         = 39
	IXOFF         = 40
	IMAXBEL       = 41
	ISIG          = 50
	ICANON        = 51
	XCASE         = 52
	ECHO          = 53
	ECHOE         = 54
	ECHOK         = 55
	ECHONL        = 56
	NOFLSH        = 57
	TOSTOP        = 58
	IEXTEN        = 59
	ECHOCTL       = 60
	ECHOKE        = 61
	PENDIN        = 62
	OPOST         = 70
	OLCUC         = 71
	ONLCR         = 72
	OCRNL         = 73
	ONOCR         = 74
	ONLRET        = 75
	CS7           = 90
	CS8           = 91
	PARENB        = 92
	PARODD        = 93
	TTY_OP_ISPEED = 128
	TTY_OP_OSPEED = 129
)

type SshSession struct {
	id          string
	session     *ssh.Session
	Stdout      *bytes.Buffer
	Stderr      *bytes.Buffer
	deadline    *time.Time
	idleTimeout int
	ch          ssh.Channel
	started     bool
}

// RFC 4254 Section 6.2.
type ptyRequestMsg struct {
	Term     string
	Columns  uint32
	Rows     uint32
	Width    uint32
	Height   uint32
	Modelist string
}

func (sc *SshSession) start() error {
	sc.started = true
	return nil
}

func (sc *SshSession) Run(cmd string) (err error) {
	sc.session.Stdout = sc.Stdout
	sc.session.Stderr = sc.Stderr
	return sc.session.Run(cmd)
}

func (sc *SshSession) RequestPty(term string, h int, w int, termmodes ssh.TerminalModes) (err error) {
	if sc.session != nil {
		return sc.session.RequestPty(term, h, w, termmodes)
	} else {
		var tm []byte
		for k, v := range termmodes {
			kv := struct {
				Key byte
				Val uint32
			}{k, v}

			tm = append(tm, ssh.Marshal(&kv)...)
		}
		tm = append(tm, tty_OP_END)
		req := ptyRequestMsg{
			Term:     term,
			Columns:  uint32(w),
			Rows:     uint32(h),
			Width:    uint32(w * 8),
			Height:   uint32(h * 8),
			Modelist: string(tm),
		}
		ok, err := sc.ch.SendRequest("pty-req", true, ssh.Marshal(&req))
		if err == nil && !ok {
			err = errors.New("ssh: pty-req failed")
		}
		return err
	}
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
	if sc.session != nil {
		return sc.session.Shell()
	} else {
		if sc.started {
			return errors.New("ssh: session already started")
		}
		ok, err := sc.ch.SendRequest("shell", true, nil)
		if err == nil && !ok {
			return errors.New("ssh: could not start shell")
		}
		if err != nil {
			return err
		}
		return sc.start()
	}
}

func (sc *SshSession) Wait() error {
	return sc.session.Wait()
}

func (sc *SshSession) Close() error {
	if sc.session != nil {
		return sc.session.Close()
	} else {
		return sc.ch.Close()
	}
}

func NewSession(conn *ssh.Client, deadline *time.Time, idleTimeout int) (ss *SshSession, err error) {
	session, err := conn.NewSession()
	if err != nil {
		return nil, err
	}
	sshSession := new(SshSession)
	sshSession.session = session
	sshSession.deadline = deadline
	sshSession.idleTimeout = idleTimeout
	sshSession.id = Rand().Hex()
	//check session timeout
	go sshSession.checkSessionTimeout()
	return sshSession, nil
}

func NewSessionWithChannel(conn *ssh.Client, ch ssh.Channel, deadline *time.Time, idleTimeout int) (ss *SshSession, err error) {
	sshSession := new(SshSession)
	sshSession.deadline = deadline
	sshSession.idleTimeout = idleTimeout
	sshSession.id = Rand().Hex()
	sshSession.ch = ch
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
			if sc.deadline != nil && time.Now().After(*sc.deadline) {
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
