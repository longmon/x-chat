/**
 ***************** X-Chat *******************
 *
 * link:x-chat.cc
 *
 ********************************************
 */

package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

var (
	Self    USER
	Runtime runtime
	server  Server
	client  Client
	err     error
)

func init() {
	var BindingPort string
	var HOST string
	var PORT string
	if len(os.Args) < 2 {
		fmt.Println("Usage:\n" + os.Args[0] + " -l [port] for server\n" + os.Args[0] + " [ip] [port] for client\n\nsite: https://x-chat.cc")
		os.Exit(1)
	}
	if len(os.Args) >= 2 && os.Args[1] == "help" {
		fmt.Println("Usage:\n" + os.Args[0] + " -l [port] for server\n" + os.Args[0] + " [ip] [port] for client\n\nsite: https://x-chat.cc")
		os.Exit(1)
	}
	flag.StringVar(&BindingPort, "l", "-1", "binding port of server")
	flag.StringVar(&HOST, "h", "", "server host")
	flag.StringVar(&PORT, "p", "", "server port")
	flag.Parse()
	if BindingPort != "-1" {
		//服务器角色
		Runtime.Mode = 0
		server.TCPAddr, err = net.ResolveTCPAddr("tcp", ":"+BindingPort)
		if err != nil {
			debugInfo(err)
			os.Exit(1)
		}
		server.Clients = make(map[string]Client, 10)
		Self.IPPort = []byte(":1:" + BindingPort)
	} else {
		Runtime.Mode = 1
		var args = flag.Args()
		if HOST == "" && len(args) >= 1 {
			HOST = args[0]
		}
		if PORT == "" && len(args) >= 2 {
			PORT = args[1]
		}
		client.RemoteAddr, err = net.ResolveTCPAddr(HOST, PORT)
		if err != nil {
			debugInfo(err)
			os.Exit(-1)
		}
		client.LastAct = time.Now().Unix()
	}
	if Runtime.Mode != 0 && Runtime.Mode != 1 {
		log.Fatalln("Rumtime error!")
	}
	os.Mkdir("./data", 0655)
}

func main() {
	if Runtime.Mode == 0 {
		server.bindAndListen()
		go server.accept()
	} else {

	}
	ReadTerminalStdin()
}
