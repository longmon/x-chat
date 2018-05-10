package main

import (
	"bufio"
	"crypto/md5"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
)

//Max block size
const MAX_BLOCK_SIZE = 1 << 13

const (
	UNKNOW_MSG_TYPE = iota //注意，不要使用这个类型，会导致HEAD的长度不是24从而读取错误

	ACK_MSG_TYPE

	TEXT_MSG_TYPE

	FILE_MSG_TYPE

	SHELL_MSG_TYPE
)

const MSG_HEAD_SIZE = 24

type client struct {
	User       *USER
	Conn       net.Conn
	RemoteAddr string
	LastAct    int64
}
type server struct {
	Addr     string
	Mutex    sync.RWMutex
	Clients  map[string]client
	Listener net.Listener
}

type runtime struct {
	Mode uint8
}

type fileIO struct {
	Fopen *os.File
}

func terminalInput() {
	rd := bufio.NewReader(os.Stdin)
	for {
		line, _, err := rd.ReadLine()
		if err != nil {
			log.Println(err)
			break
		}
		if len(line) == 0 {
			continue
		}
		iSaid(line)
		sendto(line)
	}
}

func debugLog(err error) {
	fmt.Println(err)
	fmt.Println("============================ Call Stack =======================")
	debug.PrintStack()
}

func sendto(line []byte) {
	var body MsgBody
	body.User = &Self
	body.Payload = line

	bodyData, err := proto.Marshal(&body)
	if err != nil {
		debugLog(err)
		return
	}

	var header MsgHead
	header.Typ = TEXT_MSG_TYPE
	header.Blocks = 1
	header.BodyLen = uint32(len(bodyData))
	md5string := md5.Sum(bodyData)
	header.Hash = md5string[:]

	headerData, err := proto.Marshal(&header)

	Msg := MessageB{Head: headerData, Body: bodyData}

	Msg.send(string(Self.IPPort))
	return
}

func (Svr *server) BroadCast(body []byte, unexpected string) {
	var header MsgHead
	header.Typ = TEXT_MSG_TYPE
	header.Blocks = 1
	header.BodyLen = uint32(len(body))
	md5string := md5.Sum(body)
	header.Hash = md5string[:]

	headerData, _ := proto.Marshal(&header)
	Msg := MessageB{Head: headerData, Body: body}

	Msg.send(unexpected)
}

func (Msg MessageB) send(sender string) {
	if Runtime.Mode == 0 {
		for ipport, c := range Server.Clients {
			if ipport == sender {
				continue
			}
			n, err := c.Conn.Write(Msg.Head)
			if err != nil {
				debugLog(err)
				Server.removeClient(ipport)
				continue
			}
			if n <= 0 {
				continue
			}
			_, err = c.Conn.Write(Msg.Body)
			if err != nil {
				debugLog(err)
				Server.removeClient(ipport)
				continue
			}
		}
	} else {
		n, err := Client.Conn.Write(Msg.Head)
		if err != nil {
			debugLog(err)
		}
		if n > 0 {
			Client.Conn.Write(Msg.Body)
		}
	}
}

func (Svr *server) removeClient(key string) {
	Svr.Mutex.Lock()
	Svr.Clients[key].Conn.Close()
	delete(Svr.Clients, key)
	Svr.Mutex.Unlock()
}

func (Svr *server) addClient(key string, c client) {
	Svr.Mutex.Lock()
	Svr.Clients[key] = c
	Svr.Mutex.Unlock()
}

func (Svr *server) getClient(key string) *client {
	Svr.Mutex.RLock()
	defer Svr.Mutex.RUnlock()
	if c, ok := Svr.Clients[key]; ok {
		return &c
	}
	return nil
}

func (Svr *server) bindAndListen() error {
	Svr.Listener, err = net.Listen("tcp", Svr.Addr)
	if err != nil {
		debugLog(err)
		return err
	}
	return nil
}

func (Svr *server) accept() {
	for {
		tcpConn, err := Svr.Listener.Accept()
		if err != nil {
			debugLog(err)
			os.Exit(-1)
		}
		raddr := tcpConn.RemoteAddr().String()
		remoteClient := client{User: nil, Conn: tcpConn, RemoteAddr: raddr, LastAct: time.Now().Unix()}
		Svr.Clients[raddr] = remoteClient

		sendAck(&tcpConn)

		go Svr.handleAcceptConn(&tcpConn)
	}
}

func sendAck(conn *net.Conn) {
	var header MsgHead
	var ack MsgAck

	raddrString := (*conn).RemoteAddr().String()

	ack.IPPort = []byte(raddrString)
	if Runtime.Mode == 0 {
		ack.ClientsNum = uint32(len(Server.Clients))
	} else {
		ack.ClientsNum = 0
	}
	ackBytes, err := proto.Marshal(&ack)
	if err != nil {
		debugLog(err)
		return
	}

	header.Typ = ACK_MSG_TYPE
	header.BodyLen = uint32(len(ackBytes))
	header.Blocks = 1
	md5s := md5.Sum(ackBytes)
	header.Hash = md5s[:]

	headerBytes, err := proto.Marshal(&header)

	if err != nil {
		debugLog(err)
		return
	}
	n, err := (*conn).Write(headerBytes)
	if err != nil {
		debugLog(err)
		if Runtime.Mode == 0 {
			Server.removeClient(raddrString)
		}
		return
	}
	if n > 0 {
		(*conn).Write(ackBytes)
	}
}

func (Svr *server) handleAcceptConn(conn *net.Conn) {
	for {
		if err := recvfrom(conn); err != nil {
			Svr.removeClient((*conn).RemoteAddr().String())
			debugLog(err)
			break
		}
		//todo:收到消息后还要转发出去的
	}
}

func (c *client) recvConnect() {
	for {
		if err := recvfrom(&c.Conn); err != nil {
			debugLog(err)
			break
		}
	}
}

func recvfrom(conn *net.Conn) error {
	head, err := readMsgHeader(conn)
	if err != nil {
		return err
	}

	if head.Typ == ACK_MSG_TYPE {
		ackMsgHandle(conn, &head)
	} else if head.Typ == TEXT_MSG_TYPE {
		textMsgHandle(conn, &head)
	} else if head.Typ == FILE_MSG_TYPE {
		fileMsgHandle(conn, &head)
	} else if head.Typ == SHELL_MSG_TYPE {
		shellMsgHandle(conn, &head)
	}
	return nil
}

func readMsgHeader(conn *net.Conn) (head MsgHead, err error) {

	buffer := make([]byte, MSG_HEAD_SIZE)
	n, err := (*conn).Read(buffer)

	if err != nil {
		return
	}
	err = proto.Unmarshal(buffer, &head)
	if err != nil {
		return
	}
	if n != MSG_HEAD_SIZE {
		err = errors.New("Invalid Header Size")
		return
	}
	return
}

func ackMsgHandle(conn *net.Conn, head *MsgHead) {
	buffer := make([]byte, int(head.BodyLen))
	n, err := (*conn).Read(buffer)
	if err != nil {
		debugLog(err)
		return
	}
	if n <= 0 {
		return
	}
	var ack MsgAck
	err = proto.Unmarshal(buffer, &ack)

	if err != nil {
		debugLog(err)
		return
	}
	Self.IPPort = ack.IPPort

	if Runtime.Mode == 1 {
		fmt.Printf(" \nCurrent Connected Clients(s): %d\n", ack.ClientsNum)
		readyToSaid()
	}
}

func textMsgHandle(conn *net.Conn, head *MsgHead) {
	buffer := make([]byte, int(head.BodyLen))
	n, err := (*conn).Read(buffer)
	if err != nil {
		debugLog(err)
		return
	}
	if n <= 0 {
		return
	}
	var Text MsgBody
	err = proto.Unmarshal(buffer, &Text)

	if err != nil {
		debugLog(err)
		return
	}
	rSaid(&Text)
	if Runtime.Mode == 0 {
		Server.BroadCast(buffer, string(Text.User.IPPort))
	}

}

func fileMsgHandle(conn *net.Conn, head *MsgHead) {
	ln := int(head.BodyLen)
	Blocks := int(ln/MAX_BLOCK_SIZE) - 1
	LastBlockSize := ln % MAX_BLOCK_SIZE

	var file FileBody
	var FileIo fileIO
	for i := 0; i < Blocks; i++ {
		b := make([]byte, MAX_BLOCK_SIZE)
		n, err := (*conn).Read(b)
		if err != nil {
			debugLog(err)
			return
		}
		if n <= 0 {
			continue
		}
		err = proto.Unmarshal(b, &file)
		if err != nil {
			debugLog(err)
			return
		}
		FileIo.writeFile(&file)
	}
	if FileIo.Fopen != nil {
		defer FileIo.Fopen.Close()
	}

	b := make([]byte, LastBlockSize)
	n, err := (*conn).Read(b)
	if err != nil {
		debugLog(err)
		return
	}
	if n <= 0 {
		return
	}
	err = proto.Unmarshal(b, &file)
	if err != nil {
		debugLog(err)
		return
	}
	FileIo.writeFile(&file)

	now := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("\n[\033[4m\033[1m\033[36mSystem\033[0m @%s Said]:  File Saved: ./data/%s\n", now, file.FileName)
}

func shellMsgHandle(conn *net.Conn, head *MsgHead) {}

func (fp *fileIO) writeFile(file *FileBody) bool {
	if fp.Fopen == nil {
		fileName := getAvaiFileName(file.FileName)
		fp.Fopen, err = os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0655)
		if err != nil {
			return false
		}
	}
	fp.Fopen.Write(file.Chunked)
	return true
}

func getAvaiFileName(fileName string) string {
	var i int
	for {
		_, err := os.Stat(fileName)
		if err != nil && os.IsNotExist(err) {
			return fileName
		}
		i++
		fileName = fmt.Sprintf("%s_%d", fileName, i)
	}
}

func (c *client) Dial() error {
	Conn, err := net.DialTimeout("tcp", c.RemoteAddr, time.Second*3)
	if err != nil {
		return err
	}
	c.Conn = Conn
	return nil
}

func readyToSaid() {
	fmt.Printf("\n[\033[4m\033[1m\033[36m%s\033[0m @X-Chat Saying] $ ", Self.Name)
}

func rSaid(Msg *MsgBody) {
	fmt.Printf("\033[%dA\033[K", 1)
	now := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("\n[\033[4m\033[1m\033[33m%s\033[0m @%s Said\033[0m]:\n  %s\n", Msg.User.Name, now, Msg.Payload)

	readyToSaid()
}

func iSaid(line []byte) {
	fmt.Printf("\033[%dA\033[K", 1)
	now := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("[\033[4m\033[1m\033[36m%s\033[0m @%s Said]:\n  %s\n", Self.Name, now, line)

	readyToSaid()
}
