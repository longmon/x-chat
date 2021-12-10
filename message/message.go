package message

import (
	"github.com/longmon/x-chat/user"
	"time"
	"github.com/ugorji/go/codec"
)

type Message struct {
	From user.User
	To user.User
	Payload []byte
	Ts time.Time
}

var mh = codec.MsgpackHandle{}

func (m *Message) Marshal() ([]byte, error) {
	var buf []byte
	enc := codec.NewEncoderBytes(&buf, &mh)
	err := enc.Encode(m)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func (m *Message) Unmarshal(in []byte) error {
	dec := codec.NewDecoderBytes(in, &mh)
	return dec.Decode(&m)
}