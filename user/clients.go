package user

import (
	"net"
)

type ClientDevice struct {
	UUID       string
	RemoteAddr string
	TCPConn net.Conn
}

type TCPClient struct {

}