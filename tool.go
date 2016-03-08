package gosshtool

import (
	"errors"
	"sync"
)

var (
	sshClients      map[string]*SSHClient
	sshClientsMutex sync.RWMutex
)

func init() {
	sshClients = make(map[string]*SSHClient)
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

func ExecuteCmd(cmd, hostname string) (output, errput string, err error) {
	client, err := getClient(hostname)
	if err != nil {
		return
	}
	return client.Cmd(cmd)
}
