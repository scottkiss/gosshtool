package gosshtool

import (
	"golang.org/x/crypto/ssh"
)

type keychain struct {
	signer ssh.Signer
}
