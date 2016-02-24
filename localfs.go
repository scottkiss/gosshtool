package gosshtool

import (
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"net"
)

type LocalForwardServer struct {
	LocalBindAddress string
	RemoteAddress    string
	SshServerAddress string
	SshUserName      string
	SshUserPassword  string
	SshPrivateKey    string
	tunnel           *Tunnel
}

//create tunnel
func (this *LocalForwardServer) createTunnel() {
	if this.SshUserPassword == "" && this.SshUserName == "" {
		log.Fatal("No password or private key available")
	}
	if this.SshPrivateKey != "" {
		//todo
	}

	config := &ssh.ClientConfig{
		User: this.SshUserName,
		Auth: []ssh.AuthMethod{
			ssh.Password(this.SshUserPassword),
		},
	}

	client, err := ssh.Dial("tcp", this.SshServerAddress, config)
	if err != nil {
		log.Fatal("Failed to dial: " + err.Error())
	}
	this.tunnel = &Tunnel{client}
}

func (this *LocalForwardServer) handleConnectionAndForward(conn *net.Conn) {
	sshConn, err := this.tunnel.Client.Dial("tcp", this.RemoteAddress)
	if err != nil {
		log.Fatalf("ssh client dial error:%v", err)
	}
	go localReaderToRemoteWriter(*conn, sshConn)
	go remoteReaderToLoacalWriter(sshConn, *conn)
}

func localReaderToRemoteWriter(localConn net.Conn, sshConn net.Conn) {
	_, err := io.Copy(sshConn, localConn)
	if err != nil {
		log.Fatalf("io copy error:%v", err)
	}
}

func remoteReaderToLoacalWriter(sshConn net.Conn, localConn net.Conn) {
	_, err := io.Copy(localConn, sshConn)
	if err != nil {
		log.Fatalf("io copy error:%v", err)
	}
}

var quit chan int

func (this *LocalForwardServer) Start() {
	// create tunnel
	this.createTunnel()
	ln, err := net.Listen("tcp", this.LocalBindAddress)
	if err != nil {
		log.Fatal(err)
	}
L:
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
		}
		go this.handleConnectionAndForward(&conn)

		select {
		case <-quit:
			conn.Close()
			break L
		}
	}
	log.Println("LocalForwardServer stop...")
}

func (this *LocalForwardServer) Stop() {
	err := this.tunnel.Client.Close()
	if err != nil {
		log.Fatalf("ssh client stop error:%v", err)
	}
	quit <- 1
}
