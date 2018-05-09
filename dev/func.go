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
	ACK_MSG_TYPE = iota

	TEXT_MSG_TYPE

	FILE_MSG_TYPE

	SHELL_MSG_TYPE
)

const MSG_HEAD_SIZE = 28

type client struct {
	User       *USER
	Conn       *net.TCPConn
	RemoteAddr *net.TCPAddr
	LastAct    int64
}
type server struct {
	TCPAddr  *net.TCPAddr
	Mutex    sync.RWMutex
	Clients  map[string]client
	Listener *net.TCPListener
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

func iSaid(line []byte) {
	fmt.Printf("\033[%dA\033[K", 1)
	now := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("\n[\033[36m%s \033[0m@%s Said]:\n  %s\n", Self.Name, now, line)
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
	header.Typ = 1
	header.Blocks = 1
	header.BodyLen = uint32(len(bodyData))
	md5string := md5.Sum(bodyData)
	header.Hash = md5string[:]

	headerData, err := proto.Marshal(&header)

	Msg := MessageB{Head: headerData, Body: bodyData}

	Msg.send()
	return
}

func (Msg MessageB) send() {
	if Runtime.Mode == 0 {
		for ipport, c := range Server.Clients {
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
	Svr.Listener, err = net.ListenTCP("tcp", Svr.TCPAddr)
	if err != nil {
		debugLog(err)
		return err
	}
	return nil
}

func (Svr *server) accept() {
	for {
		tcpConn, err := Svr.Listener.AcceptTCP()
		if err != nil {
			debugLog(err)
			os.Exit(-1)
		}
		raddr, _ := net.ResolveTCPAddr("tcp", tcpConn.RemoteAddr().String())
		remoteClient := client{User: nil, Conn: tcpConn, RemoteAddr: raddr, LastAct: time.Now().Unix()}
		Svr.Clients[tcpConn.RemoteAddr().String()] = remoteClient

		sendAck(tcpConn)

		go handleAcceptConn(tcpConn)
	}
}

func sendAck(conn *net.TCPConn) {
	var header MsgHead
	var ack MsgAck

	raddrString := conn.RemoteAddr().String()

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

	header.Typ = 0
	header.BodyLen = uint32(len(ackBytes))
	header.Blocks = 1
	md5s := md5.Sum(ackBytes)
	header.Hash = md5s[:]

	headerBytes, err := proto.Marshal(&header)

	if err != nil {
		debugLog(err)
		return
	}
	n, err := conn.Write(headerBytes)
	if err != nil {
		debugLog(err)
		if Runtime.Mode == 0 {
			Server.removeClient(raddrString)
		}
		return
	}
	if n > 0 {
		conn.Write(ackBytes)
	}
}

func handleAcceptConn(conn *net.TCPConn) {
	for {
		recvfrom(conn)
	}
}

func recvfrom(conn *net.TCPConn) {
	head, err := readMsgHeader(conn)
	if err != nil {
		debugLog(err)
		return
	}

	if head.Typ == ACK_MSG_TYPE {
		ackMsgHandle(conn, head.BodyLen)
	} else if head.Typ == TEXT_MSG_TYPE {
		textMsgHandle(conn, head.BodyLen)
	} else if head.Typ == FILE_MSG_TYPE {
		fileMsgHandle(conn, head.BodyLen)
	} else if head.Typ == SHELL_MSG_TYPE {
		shellMsgHandle(conn, head.BodyLen)
	}
}

func readMsgHeader(conn *net.TCPConn) (head MsgHead, err error) {

	buffer := make([]byte, MSG_HEAD_SIZE)
	n, err := conn.Read(buffer)
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

func ackMsgHandle(conn *net.TCPConn, ln uint32) {
	buffer := make([]byte, ln)
	n, err := conn.Read(buffer)
	if err != nil {
		debugLog(err)
		return
	}
	if n <= 0 {
		return
	}
	var ack MsgAck
	err = proto.Unmarshal(buffer, &ack)

	fmt.Println(ack)

	if err != nil {
		debugLog(err)
		return
	}
	Self.IPPort = ack.IPPort

	if Runtime.Mode == 1 {
		now := time.Now().Format("2006-01-02 15:04:05")
		fmt.Printf("\n[\033[36mSystem \033[0m@%s Said]:\n  Connected:%d\n", now, ack.ClientsNum)
	}
}

func textMsgHandle(conn *net.TCPConn, ln uint32) {
	buffer := make([]byte, ln)
	n, err := conn.Read(buffer)
	if err != nil {
		debugLog(err)
		return
	}
	if n <= 0 {
		return
	}
	var Text MsgBody
	err = proto.Unmarshal(buffer, &Text)

	fmt.Println(Text)

	if err != nil {
		debugLog(err)
		return
	}

	rSaid(&Text)

}

func fileMsgHandle(conn *net.TCPConn, ln uint32) {
	Blocks := int(ln/MAX_BLOCK_SIZE) - 1
	LastBlockSize := ln % MAX_BLOCK_SIZE

	var file FileBody
	var FileIo fileIO
	for i := 0; i < Blocks; i++ {
		b := make([]byte, MAX_BLOCK_SIZE)
		n, err := conn.Read(b)
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
	n, err := conn.Read(b)
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
	fmt.Printf("\n[\033[36m%System \033[0m@%s Said]:\n  File Saved: ./data/%s\n", now, file.FileName)
}

func shellMsgHandle(conn *net.TCPConn, ln uint32) {}

func rSaid(Msg *MsgBody) {
	now := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("\n[\033[36m%s \033[0m@%s Said]:\n  %s\n", Msg.User.Name, now, Msg.Payload)
}

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
