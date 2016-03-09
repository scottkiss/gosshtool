package gosshtool

import (
	"io"
	"log"
	"net"
)

type LocalForwardServer struct {
	ForwardConfig
	tunnel *Tunnel
}

//create tunnel
func (this *LocalForwardServer) createTunnel() {
	config := &SSHClientConfig{
		User:       this.SshUserName,
		Password:   this.SshUserPassword,
		Host:       this.SshServerAddress,
		Privatekey: this.SshPrivateKey,
	}
	sshclient := NewSSHClient(config)
	conn, err := sshclient.Connect()
	if err != nil {
		log.Fatal("Failed to dial: " + err.Error())
	}
	log.Println("create ssh client ok")
	this.tunnel = &Tunnel{conn}
}

func (this *LocalForwardServer) handleConnectionAndForward(conn *net.Conn) {
	sshConn, err := this.tunnel.Client.Dial("tcp", this.RemoteAddress)
	if err != nil {
		log.Fatalf("ssh client dial error:%v", err)
	}
	log.Println("create ssh connection ok")
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

func (this *LocalForwardServer) Start(call func()) {
	this.createTunnel()
	ln, err := net.Listen("tcp", this.LocalBindAddress)
	if err != nil {
		log.Fatalf("net listen error :%v", err)
	}
	defer ln.Close()
	var called bool
	for {
		if !called && call != nil {
			go call()
			called = true
		}
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
		}
		go this.handleConnectionAndForward(&conn)
		defer conn.Close()
	}
}

func (this *LocalForwardServer) Stop() {
	err := this.tunnel.Client.Close()
	if err != nil {
		log.Fatalf("ssh client stop error:%v", err)
	}
}
