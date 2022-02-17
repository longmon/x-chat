package server

import (
	"github.com/longmon/x-chat/message"
	"github.com/longmon/x-chat/user"
	"log"
	"net"
	"time"
)

type ChatServer struct {
	bindAddr string
}

func NewChatServer(addr string) *ChatServer {
	return &ChatServer{
		bindAddr: addr,
	}
}

func (s *ChatServer) ListenAndAccept() error {

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatalln(err)
	}
	var mac []string
	for _, addr := range addrs {
		mac = append(mac, addr.String())
	}

	lis, err := net.Listen("tcp", s.bindAddr)
	if err != nil {
		return  err
	}
	for {
		tcpConn, err := lis.Accept()
		if err != nil {
			log.Fatalln(err)
		}

		log.Println(tcpConn)
		sendAck(tcpConn)
	}
}

func sendAck(conn net.Conn) {
	msg := message.Message{
		From:    user.SystemUser,
		To:      user.User{
			ID:        "11",
			Name:      "223",
			LoginAt:   time.Time{},
			LastAckAt: time.Time{},
		},
		Payload: []byte("是上呻旧"),
		Ts:      time.Time{},
	}
	by, _ := msg.Marshal()
	conn.Write(by)
}

func MsgRecvHandler(conn net.Conn) {
	var buffer = make([]byte, 1024*8)
	for {
		n , err := conn.Read(buffer)
		if err != nil {
			log.Println(err)
			continue
		}

	}
}