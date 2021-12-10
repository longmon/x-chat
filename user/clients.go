package user

import "net"

type Client struct {
	UUID string
	RemoteAddr string
	TCPConn net.Conn
	User User
}
