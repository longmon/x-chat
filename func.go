package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"runtime/debug"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
)

//Max block size
const MAX_BLOCK_SIZE = 1 << 25

const (
	UNKNOW_MSG_TYPE = iota //注意，不要使用这个类型，会导致HEAD的长度不致从而读取错误

	ACK_MSG_TYPE

	TEXT_MSG_TYPE

	FILE_MSG_TYPE

	FILE_PUT_TEMP_TYPE

	FILE_RECV_ACK_TYPE

	FILE_TIP_MSG_TYPE

	FILE_GET_FILE_TYPE

	SHELL_MSG_TYPE
)

const MSG_HEAD_SIZE = 15

const TEMP_FILE_DIR = "/tmp/X-Chat-Tmp"

const ARCHIVE_HOLD_PATH = "./data"

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
	FileM
}

type MessageBob struct {
	MessageB
	User USER
}

var cmd = map[string]func(string) error{
	`<<\s*([\w.]+)`: SendFileToSvr,
	`>>\s*([\w.]+)`: requestRecvFile,
}

var MsgTypeCB = map[uint32]func(*net.Conn, *MsgHead){
	UNKNOW_MSG_TYPE:    unkownMsgType,
	ACK_MSG_TYPE:       ackMsgHandle,
	TEXT_MSG_TYPE:      textMsgHandle,
	FILE_MSG_TYPE:      fileMsgHandle,
	FILE_PUT_TEMP_TYPE: filePutTempHandle,
	FILE_RECV_ACK_TYPE: fileRecvAckHandle,
	SHELL_MSG_TYPE:     shellMsgHandle,
	FILE_TIP_MSG_TYPE:  havingFileRecvHandle,
	FILE_GET_FILE_TYPE: clientGetFileHandle,
}

var transfferFileList = map[string]FileM{}

func terminalInput() {
	rd := bufio.NewReader(os.Stdin)
	var isRegxMatch bool
	for {
		line, _, err := rd.ReadLine()
		if err != nil {
			break
		}
		if len(line) == 0 {
			continue
		}
		isRegxMatch = false
		for pat, fun := range cmd {
			regx, _ := regexp.Compile(pat)
			mat := regx.FindSubmatch(line)
			if len(mat) > 1 {
				isRegxMatch = true
				fun(string(mat[1]))
				break
			}
		}

		if !isRegxMatch {
			iSaid(line)
			sendto(line)
		}
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

	MsgBob := PacketMsgBob(bodyData, TEXT_MSG_TYPE)

	if Runtime.Mode == 0 {
		AddMsgBobQueue(MsgBob)
	} else {
		MsgBob.Send()
	}
	return
}

func (Svr *server) BroadCast() {
	for Bob := range MsgBobQueue {
		Bob.Send()
	}
}

func AddMsgBobQueue(MsgBob MessageBob) {
	MsgBobQueue <- MsgBob
}

func (MsgBob MessageBob) Send() {
	if Runtime.Mode == 0 {
		for ipport, c := range Server.Clients {
			if ipport == string(MsgBob.User.IPPort) {
				continue
			}
			n, err := c.Conn.Write(MsgBob.Head)
			if err != nil {
				debugLog(err)
				Server.removeClient(ipport)
				continue
			}
			if n <= 0 {
				continue
			}
			_, err = c.Conn.Write(MsgBob.Body)
			if err != nil {
				debugLog(err)
				Server.removeClient(ipport)
				continue
			}
		}
	} else {
		n, err := Client.Conn.Write(MsgBob.Head)
		if err != nil {
			debugLog(err)
			os.Exit(-1)
		}
		if n > 0 {
			Client.Conn.Write(MsgBob.Body)
		}
	}
}

func (MsgBob MessageBob) Sendto(conn *net.Conn) {

	if conn == nil {
		return
	}
	_, err := (*conn).Write(MsgBob.Head)
	if err != nil {
		if Runtime.Mode == 0 {
			Server.removeClient((*conn).RemoteAddr().String())
		}
		debugLog(err)
		return
	}
	_, err = (*conn).Write(MsgBob.Body)
	if err != nil {
		if Runtime.Mode == 0 {
			Server.removeClient((*conn).RemoteAddr().String())
		}
		debugLog(err)
		return
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
	ack.Name = Self.Name

	ackBytes, err := proto.Marshal(&ack)
	if err != nil {
		debugLog(err)
		return
	}

	header.Typ = ACK_MSG_TYPE
	header.BodyLen = uint32(len(ackBytes))
	header.Blocks = 1

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
		if err := recvfrom(conn); err != nil && err.Error() != "EOF" {
			Svr.removeClient((*conn).RemoteAddr().String())
			debugLog(err)
			break
		}
	}
}

func (c *client) recvConnect() {
	for {
		if err := recvfrom(&c.Conn); err != nil && err.Error() != "EOF" {
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

	if handler, ok := MsgTypeCB[head.Typ]; ok {
		handler(conn, &head)
	}
	return nil
}

func readMsgHeader(conn *net.Conn) (head MsgHead, err error) {

	buffer := make([]byte, MSG_HEAD_SIZE)
	n, err := (*conn).Read(buffer)

	if err != nil {
		return
	}

	if n != MSG_HEAD_SIZE {
		err = errors.New("Invalid Header Size")
		return
	}

	err = proto.Unmarshal(buffer, &head)
	if err != nil {
		return
	}

	return
}

func (fp *fileIO) writeFile(file *FileBody, dir string, rename bool) bool {
	if fp.Fopen == nil {
		if rename {
			file.FileName = getAvaiFileName(file.FileName)
		}
		file.FileName = fmt.Sprintf("%s/%s", dir, file.FileName)
		fp.Fopen, err = os.OpenFile(file.FileName, os.O_RDWR|os.O_CREATE, 0400)
		if err != nil {
			return false
		}
	}

	fp.Fopen.Write(file.Chunked)
	return true
}

func getAvaiFileName(fileName string) string {
	fileName = fmt.Sprintf("%s_%d", fileName, time.Now().Unix())
	return fileName
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

func sysSaid(text string) {
	fmt.Printf("\033[%dA\033[K", 1)
	now := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("\n[\033[4m\033[1m\033[35mSystem\033[0m @%s Said]:\n  %s\n", now, text)

	readyToSaid()
}

func PacketMsgBob(body []byte, MsgType uint32) MessageBob {
	var header MsgHead
	header.Typ = MsgType
	header.BodyLen = uint32(len(body))
	header.Blocks = 1

	headerData, _ := proto.Marshal(&header)

	var MsgBob MessageBob

	MsgBob.Head = headerData
	MsgBob.Body = body
	MsgBob.User = Self

	return MsgBob
}

func PackFileMsgBob(data []byte, fileName string) ([]byte, error) {
	var fileMsgBob FileBody
	fileMsgBob.FileName = fileName
	fileMsgBob.Chunked = data

	fileMsgData, err := proto.Marshal(&fileMsgBob)
	if err != nil {
		debugLog(err)
		return nil, err
	}
	return fileMsgData, nil
}
