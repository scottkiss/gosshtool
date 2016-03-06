package gosshtool

func NewSSHClient(config *SSHClientConfig) *SSHClient {
	client := new(SSHClient)
	client.Host = config.Host
	client.User = config.User
	client.Password = config.Password
	client.Privatekey = config.Privatekey
	return client
}
