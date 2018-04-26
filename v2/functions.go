package main

import (
	"crypto/md5"
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
	Typ     uint8  `Message type: 0=>ack,1=>text,2=>file`
	BodyLen int    `Length of message`
	Blocks  uint16 `Number of message blocks`
	Hash    []byte `Hash string of message`
}

//FileMsgBlock
type FileMsgBlock struct {
	BlockNo uint16 `Block number`
	Payload []byte `File block buffer`
}

//TextMsgBody TextMsg
type TextMsgBody struct {
	Payload []byte `Text message body`
}

//ActMsgBody ActMsg
type ActMsgBody struct {
	IPPort []byte `Client connected IP and Port`
	CoNum  int    `Numer of Connected client`
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

func (This *Server) bindAndListen(ip, port string) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", ip+":"+port)
	if err != nil {
		debugInfo(err)
		os.Exit(1)
	}
	This.Listener, err = net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		debugInfo(err)
		os.Exit(1)
	}
}

func (This *Server) accept() {

	for {
		tcpConn, err := This.Listener.AcceptTCP()
		if err != nil {
			debugInfo(err)
			continue
		}
		This.sendAck(tcpConn)
		go tcpConnHandle(tcpConn)
	}
}

func (This *Server) sendAck(conn *net.TCPConn) {
	var actMsg ActMsgBody
	actMsg.IPPort = []byte(conn.RemoteAddr().String())
	actMsg.CoNum = len(This.Clients)

	var Header MsgHead
	Header.Typ = 0
	Header.Blocks = 1
	Header.BodyLen = len(actMsg.IPPort) + 4
	Header.Hash = md5.Sum(actMsg.IPPort)
}

func tcpConnHandle(conn *net.TCPConn) {

}

func debugInfo(err error) {
	fmt.Println(err)
	fmt.Printf("======================== Call Stack ===================\n")
	debug.PrintStack()
}
