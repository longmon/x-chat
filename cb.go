package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"github.com/golang/protobuf/proto"
)

func unkownMsgType(conn *net.Conn, head *MsgHead) {}

func ackMsgHandle(conn *net.Conn, head *MsgHead) {
	buffer := make([]byte, int(head.BodyLen))
	n, err := (*conn).Read(buffer)
	if err != nil {
		debugLog(err)
		return
	}
	if n <= 0 {
		return
	}
	var ack MsgAck
	err = proto.Unmarshal(buffer, &ack)

	if err != nil {
		debugLog(err)
		return
	}
	Self.IPPort = ack.IPPort

	if Runtime.Mode == 1 {
		fmt.Printf(" \nCurrent Connected Clients(s): %d\n", ack.ClientsNum)
		readyToSaid()
	} else {
		remoteAddr := (*conn).RemoteAddr().String()
		var User USER
		User.Name = ack.Name
		User.IPPort = []byte(remoteAddr)
		C := Server.Clients[remoteAddr]
		C.User = &User
		Server.Clients[remoteAddr] = C
	}
}

func textMsgHandle(conn *net.Conn, head *MsgHead) {
	buffer := make([]byte, int(head.BodyLen))
	n, err := (*conn).Read(buffer)
	if err != nil {
		debugLog(err)
		return
	}
	if n <= 0 {
		return
	}
	var Text MsgBody
	err = proto.Unmarshal(buffer, &Text)

	if err != nil {
		debugLog(err)
		return
	}
	rSaid(&Text)
	if Runtime.Mode == 0 {
		MsgBob := PacketMsgBob(buffer, TEXT_MSG_TYPE)
		MsgBob.User = *Text.User
		AddMsgBobQueue(MsgBob)
	}
}

func fileMsgHandle(conn *net.Conn, head *MsgHead) {
	ln := int(head.BodyLen)
	Blocks := int(head.Blocks - 1)
	LastBlockSize := ln - Blocks*MAX_BLOCK_SIZE
	var file FileBody
	var FileIo fileIO
	for i := 0; i < Blocks; i++ {
		b := make([]byte, MAX_BLOCK_SIZE)
		n, err := (*conn).Read(b)
		if err != nil {
			debugLog(err)
			return
		}
		if n <= 0 {
			continue
		}
		err = proto.Unmarshal(b, &file)
		if err != nil {
			debugLog(err)
			return
		}
		FileIo.writeFile(&file, ARCHIVE_HOLD_PATH, false)
	}

	b := make([]byte, LastBlockSize)
	n, err := (*conn).Read(b)
	if err != nil {
		sysSaid(err.Error())
		return
	}
	if n <= 0 {
		return
	}
	err = proto.Unmarshal(b, &file)
	if err != nil {
		sysSaid(err.Error())
		return
	}
	FileIo.writeFile(&file, ARCHIVE_HOLD_PATH, false)

	FileIo.Fopen.Close()

	sysSaid(fmt.Sprintf("File Saved to %s\n", file.FileName))
}

func filePutTempHandle(conn *net.Conn, head *MsgHead) {
	ln := int(head.BodyLen)
	Blocks := int(head.Blocks - 1)
	LastBlockSize := ln - Blocks*MAX_BLOCK_SIZE

	var file FileBody
	var FileIo fileIO
	for i := 0; i < Blocks; i++ {
		b := make([]byte, MAX_BLOCK_SIZE)
		n, err := (*conn).Read(b)
		if err != nil {
			debugLog(err)
			return
		}
		if n <= 0 {
			continue
		}
		err = proto.Unmarshal(b, &file)
		if err != nil {
			debugLog(err)
			return
		}
		FileIo.writeFile(&file, TEMP_FILE_DIR, true)
	}

	b := make([]byte, LastBlockSize)
	n, err := (*conn).Read(b)
	if err != nil {
		debugLog(err)
		return
	}
	if n <= 0 {
		return
	}
	err = proto.Unmarshal(b, &file)
	if err != nil {
		debugLog(err)
		return
	}

	FileIo.FileName = file.FileName

	FileIo.writeFile(&file, TEMP_FILE_DIR, true)

	FileIo.FileTempName = file.FileName

	defer FileIo.Fopen.Close()

	var fileAck FileM

	remoteaddr := (*conn).RemoteAddr().String()
	fileAck.User = Server.Clients[remoteaddr].User
	fileAck.FileName = FileIo.FileName
	fileAck.FileTempName = FileIo.FileTempName

	transfferFileList[fileAck.FileName] = fileAck

	ackData, err := proto.Marshal(&fileAck)

	MsgBob := PacketMsgBob(ackData, FILE_RECV_ACK_TYPE)
	MsgBob.Sendto(conn)

	TipMsgBob := PacketMsgBob(ackData, FILE_TIP_MSG_TYPE)
	TipMsgBob.User = *Server.Clients[remoteaddr].User
	TipMsgBob.Send()

	MsgText := fmt.Sprintf("%s shared a file '%s'. type \033[4m\033[1m\033[35m>> %s\033[0m to recv it!", fileAck.User.Name, fileAck.FileName, fileAck.FileName)
	sysSaid(MsgText)

}

func fileRecvAckHandle(conn *net.Conn, head *MsgHead) {
	var fileAck FileM
	buf := make([]byte, head.BodyLen)
	_, err := (*conn).Read(buf)
	if err != nil {
		debugLog(err)
		return
	}
	err = proto.Unmarshal(buf, &fileAck)
	if err != nil {
		debugLog(err)
		return
	}
	isaid := fmt.Sprintf("File '%s' have been sent!", fileAck.FileName)

	sysSaid(isaid)

}

func shellMsgHandle(conn *net.Conn, head *MsgHead) {}

func havingFileRecvHandle(conn *net.Conn, head *MsgHead) {
	var fileAck FileM
	buf := make([]byte, head.BodyLen)
	_, err := (*conn).Read(buf)
	if err != nil {
		debugLog(err)
		return
	}
	err = proto.Unmarshal(buf, &fileAck)
	if err != nil {
		debugLog(err)
		return
	}
	//Client file map
	transfferFileList[fileAck.FileName] = fileAck
	MsgText := fmt.Sprintf("%s shared a file '%s'. type \033[4m\033[1m\033[35m>> %s\033[0m to recv it!", fileAck.User.Name, fileAck.FileName, fileAck.FileName)
	sysSaid(MsgText)
}

func clientGetFileHandle(conn *net.Conn, head *MsgHead) {
	var fileM FileM
	buffer := make([]byte, head.BodyLen)
	_, err := (*conn).Read(buffer)
	if err != nil {
		return
	}
	err = proto.Unmarshal(buffer, &fileM)
	if err != nil {
		sysSaid(err.Error())
		return
	}

	content, err := readFile(fileM.FileTempName)
	if err != nil {
		sysSaid(err.Error())
		return
	}
	var fileMsg FileBody
	fileMsg.FileName = fileM.FileName
	fileMsg.Chunked = content

	fileMsgBuf, err := proto.Marshal(&fileMsg)
	if err != nil {
		sysSaid(err.Error())
		return
	}
	FileMsgBob := PacketMsgBob(fileMsgBuf, FILE_MSG_TYPE)

	FileMsgBob.Sendto(conn)
}

func SendFileToSvr(file string) error {
	finfo, err := os.Stat(file)
	if err != nil && os.IsNotExist(err) {
		sysSaid("file '" + file + "' dose not exists!")
		return err
	}
	switch mode := finfo.Mode(); {
	case mode.IsRegular():
		if finfo.Size() > MAX_BLOCK_SIZE {
			sysSaid("File '" + finfo.Name() + "' is too large to be transffered! Less than 32Mb is good")
		} else {
			sendfile(file)
		}
	default:
		sysSaid("file '" + file + "' is not a regular file")
		return errors.New("not a regular file")
	}
	return nil
}

func requestRecvFile(fileName string) error {
	file, ok := transfferFileList[fileName]
	if !ok {
		sysSaid("Get file " + fileName + " failed")
		return err
	}

	if Runtime.Mode == 0 {
		copyFile(file)
		return nil
	}

	//如果是CLIENT， 需要从服务器拿回数据
	fileMdata, err := proto.Marshal(&file)
	if err != nil {
		debugLog(err)
		return err
	}

	FileGetMsg := PacketMsgBob(fileMdata, FILE_GET_FILE_TYPE)

	FileGetMsg.Sendto(&Client.Conn)

	return nil
}

func sendfile(fileName string) {
	if Runtime.Mode == 0 {
	} else {
		buffer, err := readFile(fileName)
		if err != nil {
			return
		}
		fileMsgBuf, err := PackFileMsgBob(buffer, fileName)
		if err != nil {
			return
		}
		MsgBob := PacketMsgBob(fileMsgBuf, FILE_PUT_TEMP_TYPE)
		MsgBob.Send()
	}
	return
}

func readFile(fileName string) ([]byte, error) {
	finfo, err := os.Stat(fileName)
	if err != nil && os.IsNotExist(err) {
		return nil, err
	}

	if finfo.Size() > MAX_BLOCK_SIZE {
		err = errors.New("file too large to read")
		return nil, err
	}

	fp, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, finfo.Size())

	n, err := fp.Read(buf)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		err = errors.New("file '" + finfo.Name() + "' is Empty!")
		return nil, err
	}
	return buf, nil
}

func copyFile(file FileM) {
	src := file.FileTempName
	dst := fmt.Sprintf("%s/%s", ARCHIVE_HOLD_PATH, file.FileName)

	_, err := os.Stat(src)
	if err != nil && os.IsNotExist(err) {
		sysSaid(err.Error())
		return
	}

	dstinfo, err := os.Stat(dst)

	if dstinfo != nil || os.IsExist(err) {
		os.Remove(dst)
	}
	buffer, err := ioutil.ReadFile(src)
	if err != nil {
		sysSaid(err.Error())
		return
	}
	err = ioutil.WriteFile(dst, buffer, 0400)
	if err != nil {
		sysSaid(err.Error())
		return
	}
	sysSaid(fmt.Sprintf("File Saved to %s\n", dst))
}
