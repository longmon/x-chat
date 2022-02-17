package user

import (
	"crypto/md5"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

type User struct {
	ID string
	Name string
	LoginAt time.Time
	LastAckAt time.Time
	Device ClientDevice
}

func RegisterUser() {
	fmt.Printf("Type your name:")
}

var SystemUser User

func init() {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatalln(err)
	}
	var macs []string

	for _, addr := range addrs {
		macs = append(macs, addr.String())
	}
	h := md5.New()
	h.Write([]byte(strings.Join(macs,"")))
	m := fmt.Sprintf("%X",h.Sum(nil))
	
	SystemUser = User{
		ID:        m,
		Name:      "",
		LoginAt:   time.Now(),
		LastAckAt: time.Now(),
		Device:    ClientDevice{
			UUID:       m,
			RemoteAddr: "",
			TCPConn:    nil,
		},
	}
}