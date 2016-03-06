package gosshtool

import (
	"golang.org/x/crypto/ssh"
	"log"
)

type SSHClientConfig struct {
	Host       string
	User       string
	Password   string
	Privatekey string
}

func makeConfig(user string, password string, privateKey string) (config *ssh.ClientConfig) {

	if password == "" && user == "" {
		log.Fatal("No password or private key available")
	}
	config = &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
	}
	if privateKey != "" {
		log.Println(privateKey)
		signer, err := ssh.ParsePrivateKey([]byte(privateKey))
		if err != nil {
			log.Fatalf("ssh.ParsePrivateKey error:%v", err)
		}
		clientkey := ssh.PublicKeys(signer)
		config = &ssh.ClientConfig{
			User: user,
			Auth: []ssh.AuthMethod{
				clientkey,
			},
		}
	}
	return
}
