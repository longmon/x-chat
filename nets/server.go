package nets

import (
	"log"
	"net"
)

type ChatServer struct {
	bindAddr string
}

func NewServer(addr string) *ChatServer {
	return &ChatServer{
		bindAddr: addr,
	}
}

func (s *ChatServer) ListenAndAccept() error {
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
	conn.Write([]byte())
}