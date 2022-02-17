package main

import (
	"fmt"
	"github.com/longmon/x-chat/cmd"
	"github.com/longmon/x-chat/network/client"
	"github.com/longmon/x-chat/network/server"
	"github.com/longmon/x-chat/user"
	"log"
	"net"
	"os"
	"strconv"
)

const (
	RunModeAsServer = 1
	RunModeAsClient = 2
)

var srvListenAddr = "" //聊天通信端口
var mode = 0

func main() {

	if os.Args[1] == "-s" {
		mode = RunModeAsServer
		srvListenAddr = os.Args[2]

		chatSrv := server.NewChatServer(srvListenAddr)
		go chatSrv.ListenAndAccept()


		cmd.ListenCmdInput()

	} else {
		mode = RunModeAsClient
		ip, err := net.ResolveIPAddr("ip4", os.Args[1])
		if err != nil {
			log.Fatalln(err)
		}
		port, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalln(err)
		}
		cc, err := client.Dial(ip.String(), port)
		if err != nil {
			log.Fatalln(err)
		}
		log.Printf("%#v", cc)
		user.RegisterUser()
		cmd.ListenCmdInput()
	}
}

func init() {
	if len(os.Args) < 3 {
		help()
		os.Exit(0)
	}
}

func help() {
	fmt.Printf("Usage:\n   %s -s [addr:port] for server\n   " +
		"%s [remoteAddr] [port] for client\n\nExample:\n   " +
		"run `%s -s :9001` to start a new server\n   " +
		"run `%s 192.168.1.100 9001` connect to an exists server\n\n" +
		"Link: www.x-chat.cc\n", os.Args[0], os.Args[0],os.Args[0], os.Args[0])
}