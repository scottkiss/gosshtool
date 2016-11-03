package gosshtool

import (
	crand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	mrand "math/rand"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	sshClients      map[string]*SSHClient
	sshClientsMutex sync.RWMutex
)

var seeded bool = false

var syncbufpool *sync.Pool

var uuidRegex *regexp.Regexp = regexp.MustCompile(`^\{?([a-fA-F0-9]{8})-?([a-fA-F0-9]{4})-?([a-fA-F0-9]{4})-?([a-fA-F0-9]{4})-?([a-fA-F0-9]{12})\}?$`)

type UUID [16]byte

// Hex returns a hex string representation of the UUID in xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx format.
func (this UUID) Hex() string {
	x := [16]byte(this)
	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		x[0], x[1], x[2], x[3], x[4],
		x[5], x[6],
		x[7], x[8],
		x[9], x[10], x[11], x[12], x[13], x[14], x[15])

}

// Rand generates a new version 4 UUID.
func Rand() UUID {
	var x [16]byte
	randBytes(x[:])
	x[6] = (x[6] & 0x0F) | 0x40
	x[8] = (x[8] & 0x3F) | 0x80
	return x
}
func FromStr(s string) (id UUID, err error) {
	if s == "" {
		err = errors.New("Empty string")
		return
	}

	parts := uuidRegex.FindStringSubmatch(s)
	if parts == nil {
		err = errors.New("Invalid string format")
		return
	}

	var array [16]byte
	slice, _ := hex.DecodeString(strings.Join(parts[1:], ""))
	copy(array[:], slice)
	id = array
	return
}

func MustFromStr(s string) UUID {
	id, err := FromStr(s)
	if err != nil {
		panic(err)
	}
	return id
}

func randBytes(x []byte) {
	length := len(x)
	n, err := crand.Read(x)

	if n != length || err != nil {
		if !seeded {
			mrand.Seed(time.Now().UnixNano())
		}
		for length > 0 {
			length--
			x[length] = byte(mrand.Int31n(256))
		}
	}
}

func init() {
	sshClients = make(map[string]*SSHClient)
	syncbufpool = &sync.Pool{}
	syncbufpool.New = func() interface{} {
		return make([]byte, 32*1024)
	}
}

func CopyIOAndUpdateSessionDeadline(dst io.Writer, src io.Reader, session *SshSession) (written int64, err error) {
	if wt, ok := src.(io.WriterTo); ok {
		return wt.WriteTo(dst)
	}
	if rt, ok := dst.(io.ReaderFrom); ok {
		return rt.ReadFrom(src)
	}

	buf := syncbufpool.Get().([]byte)
	defer syncbufpool.Put(buf)

	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			if session.idleTimeout > 0 {
				deadlinenew := time.Now().Add(time.Second * time.Duration(session.idleTimeout))
				session.SetDeadline(&deadlinenew)
			}
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er == io.EOF {
			break
		}
		if er != nil {
			err = er
			break
		}
	}
	return written, err
}

func NewSSHClient(config *SSHClientConfig) (client *SSHClient) {
	sshClientsMutex.RLock()
	client = sshClients[config.Host]
	if client != nil {
		return
	}
	sshClientsMutex.RUnlock()
	client = new(SSHClient)
	client.Host = config.Host
	client.User = config.User
	client.Password = config.Password
	client.Privatekey = config.Privatekey
	client.DialTimeoutSecond = config.DialTimeoutSecond
	sshClientsMutex.Lock()
	sshClients[config.Host] = client
	sshClientsMutex.Unlock()
	return client
}

func getClient(hostname string) (client *SSHClient, err error) {
	if hostname == "" {
		return nil, errors.New("host name is empty")
	}
	sshClientsMutex.RLock()
	client = sshClients[hostname]
	if client != nil {
		return client, nil
	}
	sshClientsMutex.RUnlock()
	return nil, errors.New("client not create")
}

func ExecuteCmd(cmd, hostname string) (output, errput string, currentSession *SshSession, err error) {
	client, err := getClient(hostname)
	if err != nil {
		return
	}
	return client.Cmd(cmd, nil, nil, 0)
}

func UploadFile(hostname, sourceFile, targetFile string) (stdout, stderr string, err error) {
	client, err := getClient(hostname)
	if err != nil {
		return
	}
	f, err := os.Open(sourceFile)
	if err != nil {
		return
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return
	}
	return client.TransferData(targetFile, data)
}
