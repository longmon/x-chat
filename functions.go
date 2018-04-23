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

//User 用户信息
type User struct {
	Name string
	IP   string
	Role uint8
}

//Server 服务器信息
type Server struct {
	IP       net.IP
	Port     string
	Listener *net.TCPListener
	Connects map[string]Client
}

//Client 客户端信息
type Client struct {
	RemoteAddr *net.TCPAddr
	Connected  *net.TCPConn
	LastActive int64
}

//Message 消息结构体
type Message struct {
	Type    int
	User    User
	Payload []byte
}

//运行时环境
type RunTime struct {
	Mode uint8
}

//BindAndListen 服务器端绑定与监听端口
func (Svr *Server) BindAndListen() error {
	tcpAddr, err := net.ResolveTCPAddr("tcp", Svr.IP.String()+":"+Svr.Port)
	if err != nil {
		return err
	}
	Svr.Listener, err = net.ListenTCP("tcp", tcpAddr)
	return err
}

//Accept
func (Svr *Server) Accept() {
	for {
		tcpConn, err := Svr.Listener.AcceptTCP()
		if err != nil {
			Log(err)
			continue
		}
		raddr, _ := net.ResolveTCPAddr("tcp", tcpConn.RemoteAddr().String())
		var nclient = Client{raddr, tcpConn, time.Now().Unix()}
		Svr.Connects[tcpConn.RemoteAddr().String()] = nclient

		SendAckMsg(tcpConn)
		go ReadMsgThread(tcpConn)
	}
}

func (Svr *Server) SendToMultiClient(Msg Message) {
	if len(Svr.Connects) > 0 {
		for _, client1 := range Svr.Connects {
			Msg.Send(client1.Connected)
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
		MsgHandler(&Msg)
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
		MsgHandler(&Msg)
	}
}

func SendAckMsg(conn *net.TCPConn) {
	var Msg Message
	Addr := conn.RemoteAddr().String()
	Msg.Pack(1, Self, []byte(Addr))
	Msg.Send(conn)

	var Msg1 Message
	num := fmt.Sprintf("%d", len(server.Connects))
	Msg1.Pack(3, Self, []byte(num))
	Msg1.Send(conn)
}

func (client *Client) beatHeart() {
	var Msg Message
	Msg.Pack(2, Self, nil)
	for {
		time.Sleep(time.Minute * 1)
		Msg.Send(client.Connected)
	}
}

func MsgHandler(Msg *Message) {
	switch Msg.Type {
	case 0: //正常消息
		if Msg.User.IP != Self.IP {
			Msg.Print(false)
		}
		if Self.Role == 0 {
			server.SendToMultiClient(*Msg)
		}
	case 1: //Ack包
		Self.IP = string(Msg.Payload)
	case 2: //心跳包
		var client = server.Connects[Msg.User.IP]
		client.LastActive = time.Now().Unix()
		server.Connects[Msg.User.IP] = client
	case 3: //统计数据
		fmt.Printf("\nNow we have %d people in connected!\n", Msg.Payload)
	}
}

func (Svr *Server) checkClientsIfAlive() {
	var now int64
	for {
		time.Sleep(time.Minute * 2)
		now = time.Now().Unix()
		for k, client := range Svr.Connects {
			if (now - client.LastActive) > 2*60 {
				client.Connected.Close()
				delete(Svr.Connects, k)
			}
		}
	}
}
