package main

import (
	"crypto/md5"
	"fmt"
	"net"
	"os"
	"runtime/debug"
	"sync"
	"time"
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
	LastAct int64        `User last active timestamp.Updated by heartbeat`
}

type Server struct {
	Mutex    sync.RWMutex
	Clients  map[string]Client `Client list`
	Listener *net.TCPListener  `TCP listener`
	IP       net.IP            `Binding ip`
	Port     string            `Binding port`
}

type Runtime struct {
	Mode int8 `Run mode: 1=>client, 2=>server`
}

func (Svr *Server) bindAndListen(ip, port string) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", ip+":"+port)
	if err != nil {
		debugInfo(err)
		os.Exit(1)
	}
	Svr.Listener, err = net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		debugInfo(err)
		os.Exit(1)
	}
}

func (Svr *Server) accept() {

	for {
		tcpConn, err := Svr.Listener.AcceptTCP()
		if err != nil {
			debugInfo(err)
			continue
		}
		Svr.sendAck(tcpConn)
		go Svr.tcpConnHandle(tcpConn)
	}
}

func (Svr *Server) sendAck(conn *net.TCPConn) {
	var actMsg ActMsgBody
	actMsg.IPPort = []byte(conn.RemoteAddr().String())
	actMsg.CoNum = len(Svr.Clients)

	var Header MsgHead
	Header.Typ = 0
	Header.Blocks = 1
	Header.BodyLen = len(actMsg.IPPort) + 4
	hash := md5.Sum(actMsg.IPPort)
	Header.Hash = hash[:]

	Svr.Send()
}

func (Svr *Server) addConnectedClient(conn *net.TCPConn) {
	ipport := conn.RemoteAddr().String()
	var User = USER{[]byte("::1"), []byte(ipport)}
	var c = Client{User, conn, time.Now().Unix()}
	Svr.Mutex.Lock()
	Svr.Clients[ipport] = c
	Svr.Mutex.Unlock()
}

func (Svr *Server) removeConnectedClient(ipport string) {
	Svr.Mutex.Lock()
	delete(Svr.Clients, ipport)
	Svr.Mutex.Unlock()
}

func (Svr *Server) tcpConnHandle(conn *net.TCPConn) {

}

func debugInfo(err error) {
	fmt.Println(err)
	fmt.Printf("======================== Call Stack ===================\n")
	debug.PrintStack()
}
