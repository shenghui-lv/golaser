package server

import "net"

var packHead = []byte{0xff, 0xfe, 0xff, 0xfe}

//来自设备的消息
type Pack struct {
	head    [4]byte
	length  int16
	content []byte
	crc     [2]byte
}

//消息对象,用来处理读写消息
type Msg struct {
	//消息类型
	event int
	//conn的序列
	seq int
	//mac地址
	macAddr string
	//连接
	conn net.Conn

	data []byte
}
