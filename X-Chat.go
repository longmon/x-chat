/**
* 参考 https://blog.csdn.net/kevinshq/article/details/8179252
* https://my.oschina.net/90design/blog/1613047?hmsr=studygolang.com&utm_medium=studygolang.com
* http://colobu.com/2016/10/19/Go-UDP-Programming/
* https://www.jianshu.com/p/ddb68de3238b
 */

package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
)

var (
	Self    User
	Runtime RunTime
	server  Server
	client  Client
	err     error
)

func main() {
	_init_()
	createUser()
	if Runtime.Mode == 0 {
		err = server.BindAndListen()
		if err != nil {
			Log(err)
			return
		}
		go TerminalInput()
		server.Accept()
	} else {
		err = client.Dial()
		if err != nil {
			Log(err)
			return
		}
		go TerminalInput()
		client.ReadMsg()
	}
}

func _init_() {
	var BindingPort string
	var HOST string
	var PORT string
	if len(os.Args) < 2 {
		fmt.Println("Usage:\n" + os.Args[0] + " -l port for server\n" + os.Args[0] + " ip port for client\n Link:https://github.com/longmon/X-Chat.git")
		os.Exit(1)
	}
	if len(os.Args) >= 2 && os.Args[1] == "help" {
		fmt.Println("Usage:\n" + os.Args[0] + " -l port for server\n" + os.Args[0] + " ip port for client\n Link:https://github.com/longmon/X-Chat.git")
		return
	}
	flag.StringVar(&BindingPort, "l", "-1", "binding port of server")
	flag.StringVar(&HOST, "h", "", "server host")
	flag.StringVar(&PORT, "p", "", "server port")
	flag.Parse()
	if BindingPort != "-1" {
		//服务器角色
		Runtime.Mode = 0
		server.IP = net.IPv4zero
		server.Port = BindingPort
		server.Connects = make(map[string]*net.TCPConn, 1)
		Self.Role = 0
	} else {
		Runtime.Mode = 1
		var args = flag.Args()
		if HOST == "" && len(args) >= 1 {
			HOST = args[0]
		}
		if PORT == "" && len(args) >= 2 {
			PORT = args[1]
		}
		client.RemoteAddr, err = net.ResolveTCPAddr("tcp", HOST+":"+PORT)
		if err != nil {
			log.Fatalln(err)
		}
		Self.Role = 1
	}
	if Runtime.Mode != 0 && Runtime.Mode != 1 {
		log.Fatalln("Rumtime error!")
	}
}

func createUser() {
	fmt.Printf("Type your name:")
	rd := bufio.NewReader(os.Stdin)
	line, _, err := rd.ReadLine()
	if err != nil {
		Log(err)
		os.Exit(1)
	}
	Self.Name = string(line)
	Self.IP = ":1"
	fmt.Printf("\033[%dA\r", 1)
	fmt.Printf("\n*********** \033[32mHello, \033[35m" + string(line) + "\033[32m! Thank you for using X-Chat!\033[37m**************\n")

}
