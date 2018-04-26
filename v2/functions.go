package main

import (
	"fmt"
	"net"
	"os"
	"runtime/debug"
)

const DevMode = true

//Max block size
const MAX_BLOCK_SIZE = 2 * 1024 * 1024

//USER User Info
type USER struct {
	Name   []byte
	IPPort []byte
}

//MsgHead Message header
type MsgHead struct {
	Typ     uint8    `Message type`
	BodyLen uint32   `Length of message`
	Blocks  uint16   `Number of message blocks`
	Hash    [32]byte `Hash string of message`
}

//FileMsgBlock
type FileMsgBlock struct {
	BlockNo uint16 `Block number`
	Payload []byte `File block buffer`
}

type TextMsgBody struct {
	Payload []byte `Text message body`
}

type Client struct {
	User    USER         `User info`
	Conn    *net.TCPConn `TCP Connection`
	LastAct int32        `User last active timestamp.Updated by heartbeat`
}

type Server struct {
	Clients  map[string]Client `Client list`
	Listener *net.TCPListener  `TCP listener`
	IP       net.IP            `Binding ip`
	Port     string            `Binding port`
}

type Runtime struct {
	Mode int8 `Run mode: 1=>client, 2=>server`
}

func (This *Server) BindAndListen(ip, port string) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", ip+":"+port)
	if err != nil {
		DebugInfo(err)
		os.Exit(1)
	}
	This.Listener, err = net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		DebugInfo(err)
		os.Exit(1)
	}
}

func (This *Server) Accept() {

	for {
		tcpConn, err := This.Listener.AcceptTCP()
		if err != nil {
			DebugInfo(err)
			continue
		}
		go TcpConnHandle(tcpConn)
	}
}

func TcpConnHandle(conn *net.TCPConn)

func DebugInfo(err error) {
	fmt.Println(err)
	fmt.Printf("======================== Call Stack ===================\n")
	debug.PrintStack()
}
