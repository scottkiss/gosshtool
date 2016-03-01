package main

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/scottkiss/gomagic/dbmagic"
	"github.com/scottkiss/gosshtool"
	//"io/ioutil"
	"log"
)

func dbop() {
	ds := new(dbmagic.DataSource)
	ds.Charset = "utf8"
	ds.Host = "127.0.0.1"
	ds.Port = 9999
	ds.DatabaseName = "info"
	ds.User = "root"
	ds.Password = "databasepassword"
	dbm, err := dbmagic.Open("mysql", ds)
	if err != nil {
		log.Fatal(err)
	}
	row := dbm.Db.QueryRow("select name from provinces where id=?", 1)
	var name string
	err = row.Scan(&name)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(name)
	dbm.Close()
}

func main() {
	server := new(gosshtool.LocalForwardServer)
	server.LocalBindAddress = ":9999"
	server.RemoteAddress = "127.0.0.1:3306"
	server.SshServerAddress = "180.111.111.111:22"
	server.SshUserPassword = "yourpassword"
	//buf, _ := ioutil.ReadFile("/your/home/path/.ssh/id_rsa")
	//server.SshPrivateKey = string(buf)
	server.SshUserName = "root"
	server.Start(dbop)
	defer server.Stop()
}
