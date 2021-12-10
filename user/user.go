package user

import "time"

type User struct {
	ID string
	Name string
	LoginAt time.Time
	LastAckAt time.Time
}
