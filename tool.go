package gosshtool

import (
	"sync"
)

var (
	sshClients      map[string]*SSHClient
	sshClientsMutex sync.Mutex
)

func init() {
	sshClients = make(map[string]*SSHClient)
}

func NewSSHClient(config *SSHClientConfig) (client *SSHClient) {
	sshClientsMutex.Lock()
	client = sshClients[config.Host]
	if client != nil {
		return
	}
	sshClientsMutex.Unlock()
	client = new(SSHClient)
	client.Host = config.Host
	client.User = config.User
	client.Password = config.Password
	client.Privatekey = config.Privatekey

	sshClientsMutex.Lock()
	sshClients[config.Host] = client
	sshClientsMutex.Unlock()
	return client
}
