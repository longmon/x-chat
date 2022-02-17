package client

import (
	"fmt"
	"net"
)

type Client struct {
	conn net.Conn
}

func Dial(ip string, port int) (*Client, error) {
	conn, err := net.Dial("tcp4", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return nil, err
	}
	client := &Client{
		conn: conn,
	}
	return client, nil
}

func (c *Client) Close()  {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *Client) SendByte(msg []byte) (int, error) {
	return c.conn.Write(msg)
}
