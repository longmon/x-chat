package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"net"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

/**
 * MsgHead.Typ: 0 MsgAck, 1 MsgText, 2 MsgFile
 */

const DevMode = true

//Max block size
const MAX_BLOCK_SIZE = 1 << 13

const (
	ACK_MSG_TYPE = iota

	TEXT_MSG_TYPE

	FILE_MSG_TYPE
)

const MSG_HEAD_SIZE = 28

type Client struct {
	User       USER         `User info`
	Conn       *net.TCPConn `TCP Connection`
	RemoteAddr *net.TCPAddr `Remote addr`
	LastAct    int64        `User last active timestamp.Updated by heartbeat`
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
		Svr.addConnectedClient(tcpConn)
		sendAck(tcpConn)
		go Svr.tcpConnHandle(tcpConn)
	}
}

func sendAck(conn *net.TCPConn) {
	var ackMsg MsgAck
	if Runtime.Mode == 0 {
		ackMsg.IPPort = []byte(conn.RemoteAddr().String())
		ackMsg.ClientsNum = uint32(len(Svr.Clients))
	} else {
		ackMsg.IPPort = client.RemoteAddr.String()
		ackMsg.ClientsNum = 0
	}

	ackData, err := proto.Marshal(&ackMsg)
	if err != nil {
		return
	}

	var Header MsgHead
	Header.Typ = 0
	Header.Blocks = 1
	Header.BodyLen = uint32(len(ackData))
	hash := md5.Sum(ackData)
	Header.Hash = hash[:]

	headerData, err := proto.Marshal(&Header)
	if err != nil {
		return
	}

	n, err := conn.Write(headerData)
	if err != nil {
		debugInfo(err)
		return
	}
	if n > 0 {
		conn.Write(ackData)
	}
	return
}

func (Svr *Server) addConnectedClient(conn *net.TCPConn) {
	ipport := conn.RemoteAddr().String()
	var User = USER{nil, []byte(ipport)}
	var c = Client{User, conn, conn.RemoteAddr(), time.Now().Unix()}
	Svr.Mutex.Lock()
	Svr.Clients[ipport] = c
	Svr.Mutex.Unlock()
}

func (Svr *Server) removeConnectedClient(ipport string) {
	Svr.Clients[ipport].Conn.Close()
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
	for {
		ReadMsg(conn)
	}
}

func ReadMsg(conn *net.TCPConn) {
	header, err := ReadMsgHeader(conn)
	if err != nil {
		return
	}

	if header.GetBlocks() > 0 {
		bodyLen := header.BodyLen % MAX_BLOCK_SIZE
		buffer := bytes.NewBuffer(nil)
		var boyd []byte
		var i uint32
		for i = 0; i < header.Blocks-1; i++ {
			boyd = ReadMsgBody(conn, MAX_BLOCK_SIZE)
			if boyd != nil {
				buffer.Write(boyd)
			}
		}
		boyd = ReadMsgBody(conn, bodyLen)
		if boyd != nil {
			buffer.Write(boyd)
		}

		switch header.Typ {
		case ACK_MSG_TYPE:
			handleAckMsg(buffer.Bytes())
		case TEXT_MSG_TYPE:
			handleTextMsg(buffer.Bytes())
		case FILE_MSG_TYPE:
			handleFileMsg(buffer.Bytes())
		}
	}
}

func ReadMsgBody(conn *net.TCPConn, size uint32) []byte {
	buf := make([]byte, size)
	n, err := conn.Read(buf)
	if err != nil {
		if Runtime.Mode == 0 {
			server.removeConnectedClient(conn.RemoteAddr().String())
		}
		debugInfo(err)
		return nil
	}
	if n > 0 {
		return buf
	}
	return nil
}

func ReadMsgHeader(conn *net.TCPConn) (header MsgHead, err error) {
	head := make([]byte, MSG_HEAD_SIZE)
	n, err := conn.Read(head)
	if err != nil {
		if Runtime.Mode == 0 {
			server.removeConnectedClient(conn.RemoteAddr().String())
		}
		debugInfo(err)
		return header, err
	}
	if n != MSG_HEAD_SIZE {
		return header, errors.New("Error MsgHead size")
	}
	err = proto.Unmarshal(head, &header)
	if err != nil {
		return header, err
	}
	return header, nil
}

func handleAckMsg(Msg []byte) {
	var msgAck MsgAck
	err := proto.Unmarshal(Msg, &msgAck)
	if err != nil {
		return
	}
	CoNum := msgAck.ClientsNum
	Self.IPPort = msgAck.IPPort
	fmt.Printf("============= Connected:%d===============", CoNum)
}

func handleTextMsg(Msg []byte) {
	var text MsgBody
	err := proto.Unmarshal(Msg, &text)
	if err != nil {
		return
	}
	Echoz(text.GetPayload())
}

func handleFileMsg(Msg []byte) {
	var fileBuffer FileBody

	err := proto.Unmarshal(Msg, &fileBuffer)
	if err != nil {
		return
	}
	file := createFile(fileBuffer.FileName)
	defer file.Close()
	_, err = file.Write(fileBuffer.Chunked)
	if err != nil {
		return
	}
	Echoz([]byte("File Saved as ./data/" + fileBuffer.FileName))
}

func createFile(fileName string) *os.File {
	var FilePtr *os.File
	_, err := os.Stat(fileName)
	if err != nil && err.Error() == os.ErrNotExist.Error() {
		FilePtr, err = os.Create(fileName)
		if err != nil {
			return FilePtr
		}
	} else {
		var fno int
		for {
			fno++
			fileName = fmt.Sprintf("%s_%d", fileName, fno)
			_, err = os.Stat(fileName)
			if err != nil && err.Error() == os.ErrNotExist.Error() {
				break
			}
		}
		FilePtr, err = os.Create(fileName)
		return FilePtr
	}
	return nil
}

func ReadTerminalStdin() {
	for {
		Rd := bufio.NewReader(os.Stdin)
		line, _, err := Rd.ReadLine()
		if err != nil {
			debugInfo(err)
			break
		}
		Echoz(line)
		if Runtime.Mode == 0 {

		}
		SendMsg(line)
	}
}

func SendMsg(line []byte) {
	var Msg MessageB
	var header MsgHead
	var body MsgBody

	header.Typ = TEXT_MSG_TYPE
	header.Blocks = 1
	header.BodyLen = uint32(len(line))
	md5byte := md5.Sum(line)
	header.Hash = md5byte[:]

	body.User = Self
	body.Payload = line

	Msg.Head, err = proto.Marshal(&header)
	if err != nil {
		debugInfo(err)
		return
	}
	Msg.Body, err = proto.Marshal(&body)
	if err != nil {
		debugInfo(err)
		return
	}

	if Runtime.Mode == 0 {
		for ipport, Rclient := range server.Clients {
			if err := Msg.Send(Rclient.Conn); err != nil {
				server.removeConnectedClient(ipport)
			}
		}
	} else {
		if err := Msg.Send(client.Conn); err != nil {
			Echoz([]byte("Send Msg error"))
		}
	}
}

func (M MessageB) Send(conn *net.TCPConn) error {
	n, err := conn.Write(M.Head)
	if err != nil {
		debugInfo(err)
		return err
	}
	if n <= 0 {
		return errors.New("Sent 0 size Msg head")
	}
	n, err = conn.Write(M.Body)
	if err != nil {
		debugInfo(err)
		return err
	}
	if n <= 0 {
		return errors.New("Sending 0 size Msg body")
	}
	return nil
}

func (c *Client) connect() {
	c.Conn, err = net.DialTCP("tcp", nil, c.RemoteAddr)
	if err != nil {
		debugInfo(err)
		os.Exit(-1)
	}

}

func debugInfo(err error) {
	fmt.Println(err)
	fmt.Printf("======================== Call Stack ===================\n")
	debug.PrintStack()
}

func Echoz(text []byte) {
	fmt.Printf("===Recved Message: %s\n", text)
}
