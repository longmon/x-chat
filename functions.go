package main

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"os"
	"runtime/debug"
	"time"
)

type User struct {
	Name string
	IP   string
	Role uint8
}

type Server struct {
	IP       net.IP
	Port     string
	Listener *net.TCPListener
	Connects map[string]*net.TCPConn
}
type Client struct {
	RemoteAddr *net.TCPAddr
	Connected  *net.TCPConn
}
type Message struct {
	Type    int
	User    User
	Payload []byte
}

type RunTime struct {
	Mode uint8
}

func (Svr *Server) BindAndListen() error {
	tcpAddr, err := net.ResolveTCPAddr("tcp", Svr.IP.String()+":"+Svr.Port)
	if err != nil {
		return err
	}
	Svr.Listener, err = net.ListenTCP("tcp", tcpAddr)
	return err
}

func (Svr *Server) Accept() {
	for {
		tcpConn, err := Svr.Listener.AcceptTCP()
		if err != nil {
			Log(err)
			continue
		}
		Svr.Connects[tcpConn.RemoteAddr().String()] = tcpConn
		SendAckMsg(tcpConn)
		go ReadMsgThread(tcpConn)
	}
}

func (Svr *Server) SendToMultiClient(Msg Message) {
	if len(Svr.Connects) > 0 {
		for _, conn := range Svr.Connects {
			Msg.Send(conn)
		}
	}
}

func (Msg Message) Send(conn *net.TCPConn) error {
	var network bytes.Buffer
	enc := gob.NewEncoder(&network)
	err := enc.Encode(Msg)
	if err != nil {
		Log(err)
		return err
	}
	_, err = conn.Write(network.Bytes())
	if err != nil {
		Log(err)
		return err
	}
	return nil
}

func (Msg *Message) Read(rd io.Reader) error {
	dec := gob.NewDecoder(rd)
	err := dec.Decode(Msg)
	if err != nil {
		return err
	}
	return nil
}
func (Msg *Message) Pack(typ int, user User, payload []byte) {
	Msg.Type = typ
	Msg.User = user
	Msg.Payload = payload
}

func (Msg *Message) Print(isSender bool) {
	var name string
	if isSender {
		fmt.Printf("\033[%dA\033[K", 1)
		name = "\033[35mYou\033[0m"
	} else {
		name = string(Msg.User.Name)
		name = "\033[32m" + name + "\033[0m"
	}

	fmt.Printf("\r[%s@%s Said]:\n  ", name, time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println(string(Msg.Payload))

	fmt.Printf("\n[" + string(Self.Name) + "@X-Chat Saying]:")
}

func (client *Client) Dial() error {
	client.Connected, err = net.DialTCP("tcp", nil, client.RemoteAddr)
	if err != nil {
		return err
	}
	return nil
}
func (client *Client) ReadMsg() {
	for {
		var Msg Message
		err := Msg.Read(client.Connected)
		if err != nil {
			break
		}
		if Msg.Type == 0 {
			if Msg.User.IP != Self.IP {
				Msg.Print(false)
			}
		} else if Msg.Type == 1 {
			Self.IP = string(Msg.Payload)
		}
	}
}

func TerminalInput() {
	rd := bufio.NewReader(os.Stdin)
	fmt.Printf("\n[" + string(Self.Name) + "@X-Chat Saying]:")
	for {
		line, _, err := rd.ReadLine()
		if err != nil {
			Log(err)
			continue
		}
		if len(line) == 0 {
			continue
		}
		var Msg Message
		Msg.Pack(0, Self, line)
		Msg.Print(true)
		if Runtime.Mode == 0 {
			server.SendToMultiClient(Msg)
		} else {
			Msg.Send(client.Connected)
		}
	}
}

func (Msg Message) sizeOf() int {
	var size int
	size += len(Msg.User.Name)
	size += len(Msg.User.IP)
	size += 1
	size += len(Msg.Payload)
	return size
}

func Log(err error) {
	fmt.Println(err)
	debug.PrintStack()
}

func ReadMsgThread(conn *net.TCPConn) {
	rd := bufio.NewReader(conn)
	var Msg Message
	for {
		err := Msg.Read(rd)
		if err != nil {
			delete(server.Connects, conn.RemoteAddr().String())
			continue
		}
		if Msg.User.IP != Self.IP {
			Msg.Print(false)
		}
		server.SendToMultiClient(Msg)
	}
}

func SendAckMsg(conn *net.TCPConn) {
	var Msg Message
	Addr := conn.RemoteAddr().String()
	Msg.Pack(1, Self, []byte(Addr))
	Msg.Send(conn)
}
