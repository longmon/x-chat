/**
 ***************** X-Chat *******************
 *
 * link:x-chat.cc
 *
 ********************************************
 */

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"time"
)

var (
	Self        USER
	Runtime     runtime
	Server      server
	Client      client
	err         error
	MsgBobQueue chan MessageBob
	FileRecv    chan int
)

func init() {

	if len(os.Args) < 2 || os.Args[1] == "help" {
		help()
		os.Exit(-1)
	}

	if len(os.Args) > 2 && os.Args[1] == "-l" {
		BindingPort := os.Args[2]

		//服务器角色
		Runtime.Mode = 0
		Server.Addr = ":" + BindingPort
		if err != nil {
			debugLog(err)
			os.Exit(1)
		}
		Server.Clients = make(map[string]client, 10)
		MsgBobQueue = make(chan MessageBob, 64)

		Self.IPPort = []byte(":" + BindingPort)
	} else {
		Runtime.Mode = 1
		if len(os.Args) > 2 {
			RemoteHost := os.Args[1]
			RemotePort := os.Args[2]
			Client.RemoteAddr = RemoteHost + ":" + RemotePort
		}
		Client.LastAct = time.Now().Unix()
	}
	if Runtime.Mode != 0 && Runtime.Mode != 1 {
		log.Fatalln("Rumtime error!")
		os.Exit(-1)
	}
	os.Mkdir("./data", 0655)
}

func main() {

	signup()

	if Runtime.Mode == 0 {
		readyToSaid()
		go terminalInput()
		go Server.BroadCast()
		Server.bindAndListen()
		Server.accept()
	} else {
		err = Client.Dial()
		if err != nil {
			debugLog(err)
			os.Exit(-1)
		}
		go Client.recvConnect()
		terminalInput()
	}
}

func help() {
	fmt.Println("Usage:\n" + os.Args[0] + " -l [port] for server\n" + os.Args[0] + " [ip] [port] for client\n\nLink:https://x-chat.cc")
}

func signup() {
	fmt.Printf("Type your name:")
	Rd := bufio.NewReader(os.Stdin)
	name, _, err := Rd.ReadLine()
	if err != nil {
		debugLog(err)
		os.Exit(-1)
	}
	Self.Name = name
	Self.IPPort = []byte(":1")
	fmt.Printf("\033[%dA\033[K", 1)
	fmt.Printf("\n*********** \033[32mHello, \033[1m\033[36m\033[4m%s\033[0m\033[32m! Thank you for using X-Chat!\033[0m**************\n", name)
}
