package main

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

const DevMode = true

//Max block size
const MAX_BLOCK_SIZE = 2 << 20

const (
	ACK_MSG_TYPE = iota

	TEXT_MSG_TYPE

	FILE_MSG_TYPE
)

//USER User Info
type USER struct {
	Name   []byte
	IPPort []byte
}

const MSG_HEAD_SIZE = 33

//MsgHead Message header sizeof 33byte
type MsgHead struct {
	Typ     uint8    `Message type: 0=>ack,1=>text,2=>file`
	Blocks  uint16   `Number of message blocks`
	BodyLen int      `Length of message`
	Hash    [16]byte `Hash string of message`
}

//MsgBlock
type MsgBlock struct {
	BlockNo uint16 `Block number`
	Payload []byte `File block buffer`
}

type Message struct {
	Head MsgHead
	Body MsgBlock
}

//ActMsgBody ActMsg
type ActBody struct {
	CoNum  uint16 `Numer of Connected client`
	IPPort []byte `Client connected IP and Port`
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
	TCPAddr  *net.TCPAddr      `TCP Addr`
}

type runtime struct {
	Mode int8 `Run mode: 1=>client, 2=>server`
}

func (Svr *Server) bindAndListen() {
	Svr.Listener, err = net.ListenTCP("tcp", Svr.TCPAddr)
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
	var actMsg ActBody
	actMsg.IPPort = []byte(conn.RemoteAddr().String())
	actMsg.CoNum = uint16(len(Svr.Clients))

	var Header MsgHead
	Header.Typ = 0
	Header.Blocks = 1
	Header.BodyLen = len(actMsg.IPPort) + 4
	hash := md5.Sum(actMsg.IPPort)
	Header.Hash = hash

	head := bytes.NewBuffer(nil)
	binary.Write(head, binary.LittleEndian, Header.Typ)
	binary.Write(head, binary.LittleEndian, Header.Blocks)
	binary.Write(head, binary.LittleEndian, Header.BodyLen)
	binary.Write(head, binary.LittleEndian, Header.Hash)
	if head.Len() <= 0 {
		return
	}
	n, err := conn.Write(head.Bytes())
	if err != nil {
		debugInfo(err)
		return
	}
	if n > 0 {
		body := bytes.NewBuffer(nil)
		binary.Write(body, binary.LittleEndian, actMsg.CoNum)
		binary.Write(body, binary.LittleEndian, actMsg.IPPort)
		conn.Write(body.Bytes())
	}
	return
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

func (Svr *Server) getConnectedClient(ipport string) Client {
	Svr.Mutex.RLock()
	client := Svr.Clients[ipport]
	Svr.Mutex.RUnlock()
	return client
}

func (Svr *Server) tcpConnHandle(conn *net.TCPConn) {

}

func (Svr *Server) readMsg(conn *net.TCPConn) {
	for {
		header, err := handleMsgHeader(conn)
		if err != nil {
			continue
		}
		boyd := readsizefrom(conn, header.BodyLen)
		if boyd == nil {
			continue
		}
		switch header.Typ {
		case ACK_MSG_TYPE:
			Self.handleAckBody(boyd)
		case TEXT_MSG_TYPE:
			//todo
		}
	}
}

func readsizefrom(conn *net.TCPConn, size int) []byte {
	buf := make([]byte, size)
	n, err := conn.Read(buf)
	if err != nil {
		debugInfo(err)
		return nil
	}
	if n > 0 {
		return buf
	}
	return nil
}

func handleMsgHeader(conn *net.TCPConn) (*MsgHead, error) {
	head := make([]byte, MSG_HEAD_SIZE)
	n, err := conn.Read(head)
	if err != nil {
		debugInfo(err)
		return nil, err
	}
	if n != MSG_HEAD_SIZE {
		return nil, errors.New("Error MsgHead size")
	}
	var header MsgHead
	header.Typ = uint8(head[0])
	header.Blocks = binary.LittleEndian.Uint16(head[1:2])
	header.BodyLen = int(binary.LittleEndian.Uint32(head[2:5]))
	for k, v := range head[6:] {
		header.Hash[k] = v
	}
	return &header, nil
}

func (client *Client) handleAckBody(body []byte) {
	CoNum := binary.LittleEndian.Uint16(body[:1])
	Self.IPPort = body[2:]
	fmt.Printf("============= Connected:%d===============", CoNum)
}

func (M Message) send(conn *net.TCPConn) {}

func debugInfo(err error) {
	fmt.Println(err)
	fmt.Printf("======================== Call Stack ===================\n")
	debug.PrintStack()
}
