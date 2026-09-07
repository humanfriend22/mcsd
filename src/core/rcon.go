package core

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	. "mcsd/utils"
)

const (
	rconTypeAuth     = 3
	rconTypeCommand  = 2
	rconTypeResponse = 0

	rconMaxPacketSize = 4110
	rconDialTimeout   = 5 * time.Second
	rconCmdTimeout    = 10 * time.Second
)

type RCONClient struct {
	conn net.Conn
	id   int32
}

type rconPacket struct {
	id    int32
	ptype int32
	body  []byte
}

func DialRCON(addr, password string) (*RCONClient, error) {
	conn, err := net.DialTimeout("tcp", addr, rconDialTimeout)
	if err != nil {
		return nil, &InternalError{Message: fmt.Sprintf("rcon dial %s: %s", addr, err.Error())}
	}
	client := &RCONClient{conn: conn}
	if err := client.auth(password); err != nil {
		conn.Close()
		return nil, err
	}
	return client, nil
}

func (client *RCONClient) Send(cmd string) (string, error) {
	client.id++
	client.conn.SetDeadline(time.Now().Add(rconCmdTimeout))
	if err := client.send(rconPacket{id: client.id, ptype: rconTypeCommand, body: []byte(cmd)}); err != nil {
		return "", &InternalError{Message: fmt.Sprintf("rcon send: %s", err.Error())}
	}
	resp, err := client.recv()
	client.conn.SetDeadline(time.Time{})
	if err != nil {
		return "", &InternalError{Message: fmt.Sprintf("rcon recv: %s", err.Error())}
	}
	return string(resp.body), nil
}

func (client *RCONClient) Close() error {
	return client.conn.Close()
}

func (client *RCONClient) auth(password string) error {
	client.id = 1
	if err := client.send(rconPacket{id: client.id, ptype: rconTypeAuth, body: []byte(password)}); err != nil {
		return &InternalError{Message: fmt.Sprintf("rcon auth send: %s", err.Error())}
	}
	client.conn.SetDeadline(time.Now().Add(rconDialTimeout))
	resp, err := client.recv()
	client.conn.SetDeadline(time.Time{})
	if err != nil {
		return &InternalError{Message: fmt.Sprintf("rcon auth recv: %s", err.Error())}
	}
	if resp.id == -1 {
		return &InternalError{Message: "rcon auth failed: wrong password"}
	}
	return nil
}

func (client *RCONClient) send(packet rconPacket) error {
	size := int32(4 + 4 + len(packet.body) + 2)
	buf := make([]byte, 4+size)
	binary.LittleEndian.PutUint32(buf[0:], uint32(size))
	binary.LittleEndian.PutUint32(buf[4:], uint32(packet.id))
	binary.LittleEndian.PutUint32(buf[8:], uint32(packet.ptype))
	copy(buf[12:], packet.body)
	_, err := client.conn.Write(buf)
	return err
}

func (client *RCONClient) recv() (rconPacket, error) {
	var size int32
	if err := binary.Read(client.conn, binary.LittleEndian, &size); err != nil {
		return rconPacket{}, &InternalError{Message: fmt.Sprintf("read size: %s", err.Error())}
	}
	if size < 10 || size > rconMaxPacketSize {
		return rconPacket{}, &InternalError{Message: fmt.Sprintf("invalid packet size: %d", size)}
	}
	data := make([]byte, size)
	if _, err := io.ReadFull(client.conn, data); err != nil {
		return rconPacket{}, &InternalError{Message: fmt.Sprintf("read packet: %s", err.Error())}
	}
	packet := rconPacket{
		id:    int32(binary.LittleEndian.Uint32(data[0:])),
		ptype: int32(binary.LittleEndian.Uint32(data[4:])),
	}
	if end := size - 10; end > 0 {
		packet.body = data[8 : 8+end]
	}
	return packet, nil
}
