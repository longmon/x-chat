package main

import (
	"flag"
	"log"
	"net"
)

var srvListenAddr = "" //聊天通信端口

func main() {

	if srvListenAddr != "" {
		ipAddr := net.ParseIP(srvListenAddr)
		log.Println(ipAddr)
	}
}

func init() {
	flag.StringVar(&srvListenAddr, "s", "","作为服务器监听的地址")
	flag.Parse()
}
