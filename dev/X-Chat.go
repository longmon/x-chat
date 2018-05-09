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
	"net"
	"os"
	"time"
)

var (
	Self    USER
	Runtime runtime
	Server  server
	Client  client
	err     error
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
		Server.TCPAddr, err = net.ResolveTCPAddr("tcp", ":"+BindingPort)
		if err != nil {
			debugLog(err)
			os.Exit(1)
		}
		Server.Clients = make(map[string]client, 10)
		Self.IPPort = []byte(":" + BindingPort)
	} else {
		Runtime.Mode = 1
		if len(os.Args) > 2 {
			RemoteHost := os.Args[1]
			RemotePort := os.Args[2]
			Client.RemoteAddr, err = net.ResolveTCPAddr("tcp", RemoteHost+":"+RemotePort)
			if err != nil {
				debugLog(err)
				os.Exit(-1)
			}
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
		go terminalInput()
		Server.bindAndListen()
		Server.accept()
	} else {
		err := Client.Dial()
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
	fmt.Printf("\n*********** \033[32mHello, \033[35m%s\033[32m! Thank you for using X-Chat!\033[37m**************\n", name)
}
