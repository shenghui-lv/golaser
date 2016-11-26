package server

import (
	"goLasereggAgent/util"
	"log"
	"net"
	"time"

	//"strconv"
	//"errors"
	//"io/ioutil"
	//	"math"
	// "fmt"
	//	"github.com/xladykiller/gotest/util"
)

const (
	conn_timeout_seconds    = 600
	state_init              = iota
	state_wait_for_key_resp = iota
	state_recv_data         = iota
)

// var rompath0 string = "App0.bin"
// var rompath1 string = "App1.bin"

const (
	event_close = iota
	event_sendKey
	event_rev
	event_send
)

/*
var event_close int = 1
var event_sendKey int = 2
var event_rev int = 3
var event_send int = 4
*/

func newCloseMsg(seq int, macAddr string, conn net.Conn) Msg {
	msg := Msg{event_close, seq, macAddr, conn, nil}
	return msg
}

func newSendMsg(event int, seq int, macAddr string, conn net.Conn, data []byte) Msg {
	msg := Msg{event, seq, macAddr, conn, data}
	return msg
}

// channel for receiving data
var revChan = make(chan Msg, 4000)

// channel for sending data
var sendChan = make(chan Msg, 4000)

//处理连接, 传给revChan的消息必须带上seq
func handleConnection(seq int, conn net.Conn) {

	defer func() {
		if err := recover(); err != nil {
			log.Println("handleConnection() crashed (", seq, "][", conn.RemoteAddr().String(), "][", err, "]")
		}
	}()

	// variables in this goroutine
	macAddr := ""
	key := ""
	state := state_init
	rawData := make([]byte, 2048)
	remainingData := make([]byte, 2048)
	recvBuf := make([]byte, 2048)
	packet := make([]byte, 2048)
	var baseMsg Msg
	baseMsg.conn = conn
	baseMsg.seq = seq
	// end of variable definition

	defer func() {
		log.Println("close (", seq, ") [", macAddr, "][", conn.RemoteAddr().String(), "]")
		conn.Close()
		closeMsg := newCloseMsg(seq, macAddr, conn)
		sendChan <- closeMsg
		// d := recover()
		// log.Println("panic:", d)
	}()

	log.Println("open  (", seq, ") {", conn.RemoteAddr().Network(), "}[", conn.RemoteAddr().String(), "]")
	conn.SetReadDeadline(time.Now().Add(20 * time.Second))

	for {
		// reading from Laseregg
		// block if no stream
		nLen, err := conn.Read(recvBuf)

		// shut goroutine when error occured
		if err != nil {
			log.Println("ERR (", seq, ") [", err.Error(), "] at state (", state, ")")
			return
		}

		// unpack packet
		rawData = append(remainingData, recvBuf[:nLen]...)
		packet, remainingData, err = Unpack(rawData)
		// -- log.Printf("rawData:%x\n", rawData)
		// -- log.Printf("read nLen:%x, pack:%x\n", nLen, packet)

		// if wrong CRC, err can tell
		if err != nil && state == state_recv_data {
			log.Println("Data packet CRC failed")
			SendClose(newCloseMsg(seq, macAddr, conn))
			continue
		}

		// no packet
		if packet == nil {
			// log.Println("No packet received (should never be printed)")
			continue
		}

		if isHelloMsg(packet) {
			// log.Println("Hello from Laseregg", seq)
			time.Sleep(1 * time.Second)
			conn.SetReadDeadline(time.Now().Add(20 * time.Second))
			key = util.RandomAscii(32)
			SendKey(baseMsg, key)
			state = state_wait_for_key_resp
		} else {

			switch state {

			case state_init:
				log.Printf("Handshake ERR (", seq, "), not hello msg from Laseregg")

			case state_wait_for_key_resp:
				// check if key matches
				leMac, _, check_ret := checkAndFetchMac(key, packet)
				sendMsg := baseMsg
				sendMsg.event = event_send
				if check_ret {
					log.Println("Authorized (", seq, ") [", leMac, "]")
					macAddr = leMac
					sendMsg.macAddr = macAddr
					sendMsg.conn = conn
					conn.SetReadDeadline(time.Now().Add(conn_timeout_seconds * time.Second))
					time.Sleep(300 * time.Millisecond)
					SendWelcome(sendMsg)
					state = state_recv_data
					handshakeMsg := Msg{event_rev, seq, macAddr, conn, packet}
					revChan <- handshakeMsg
				} else {
					log.Println("Authorizing Failed (", seq, ") [", leMac, "]")
					sendMsg.conn = conn
					SendFail(sendMsg)
					SendClose(sendMsg)
				}

			case state_recv_data:
				if isHelloMsg(packet) {
				}
				conn.SetReadDeadline(time.Now().Add(conn_timeout_seconds * time.Second))
				revMsg := baseMsg
				revMsg.event = event_rev
				revMsg.macAddr = macAddr
				revMsg.data = packet
				revChan <- revMsg

			default:
				log.Println("Wrong State")
				sendMsg := baseMsg
				sendMsg.conn = conn
				SendFail(sendMsg)
				SendClose(sendMsg)
				return
			}
		}
	}

}

//所有连接收到的消息，统一路由
func routeRev() {
	for msg := range revChan {
		// log.Println("routeRev req:", msg.seq)
		action, err := GetAction(msg.data)
		if err != nil {
			// log.Println(err.Error())
			continue
		}
		switch BytesToUint16(action) {

		case 1:
			// update handshake time
			// log.Printf("Handshake:[%x]", msg.data)
			// go HandshakeDB(msg)
			//			if laseregg_database != nil {
			go Handshake_LE_DB(msg)
			//		}

		case 2:
			// insert data into database
			// log.Printf("Insert into Database:[%x]\n", msg.data)
			//go SaveDataDB(msg)
			//	if laseregg_database != nil {
			go SaveData_LE_DB(msg)
			go SendCali_LE_DB(msg)
			//} else {
			//go SendCaliDB(msg)
			//}

		case 3:
			//发送给设备的action，非接收action
			// log.Printf("发送给设备的action，非接收action, data:%x\n", msg.data)
		case 5:
			//这个条取消了
			// log.Printf("收到OTA回复:%x\n", msg.data)
			// DoAction5(msg)
		}
	}
}

//路由所有要发送的消息
func routeSend() {
	tmpConnMap := make(map[int]chan []byte)
	connMap := make(map[string]chan []byte)
	for msg := range sendChan {
		macAddrPort := msg.macAddr + msg.conn.RemoteAddr().String()
		// log.Println("remoteAddr:", macAddrPort)
		//处理要发送的消息
		bufChan := connMap[macAddrPort]
		if bufChan == nil {
			bufChan = tmpConnMap[msg.seq]
			if bufChan != nil {
				// log.Println("从seq获取bufChan")
			}
			//需要移动到connMap
			if bufChan != nil && msg.macAddr != "" {
				// log.Println("bufChan从tmpConnMap移动到connMap at macAddrPort:", macAddrPort, ",seq:", msg.seq)
				connMap[macAddrPort] = bufChan
				delete(tmpConnMap, msg.seq)
			}
		} else {
			// log.Println("从macAddr获取bufChan")
		}

		//如果写channel不存在，则新建，并执行
		if bufChan == nil && msg.conn != nil {
			bufChan = make(chan []byte, 10)
			if msg.macAddr == "" {
				// log.Println("添加bufChan as seq:", msg.seq)
				tmpConnMap[msg.seq] = bufChan
			} else {
				// log.Println("添加bufChan as macAddrPort:", macAddrPort)
				connMap[macAddrPort] = bufChan
			}
			go doSend(msg.conn, bufChan)
		}

		if bufChan == nil {
			continue
		}
		//关闭连接
		if msg.event == event_close {
			// log.Println("delete bufChan at seq:", msg.seq, ", at macAddr:", msg.macAddr)
			close(bufChan)
			delete(connMap, macAddrPort)
			delete(tmpConnMap, msg.seq)
			continue
		}
		//通知写goroutine向conn写数据
		bufChan <- msg.data

	}
}

//真正的发送goroutine
func doSend(conn net.Conn, bufChan <-chan []byte) {
	defer func() {
		// log.Println("doSend关闭连接")
		conn.Close()
	}()
	// log.Println("启动doSend")
	for data := range bufChan {
		// log.Println("doSend获取到数据")
		// log.Printf("发送数据:%x", data)
		conn.Write(data)
	}
}

// start server listening
func Server(port string) {
	if port == "" {
		log.Println("Please specify port")
		return
	}
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Println("Error listening:", err.Error())
		return
	}
	defer func(listen net.Listener) {
		log.Println("Server listening close")
		listen.Close()
	}(listener)

	log.Println("Initializing database connection...")

	if !InitLasereggDatabaseFromAWS() {
		log.Println("Cannot establish Laseregg database connection. Exit")
		return
	}
	//	if !InitLasereggDatabase() {
	//		log.Println("Cannot establish Laseregg database connection. Exit")
	//		return
	//	}
	log.Println("Database connection initialized")
	log.Println("Starting server from ", port)

	log.Println("Ready to receive")
	go routeRev()
	log.Println("Ready to send")
	go routeSend()
	log.Println("---------------------------------------------")

	var seq int = 0
	// server main loop
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("(", seq, ") Error accepting: <", err.Error(), ">")
			// return
		}
		seq++
		seq = seq % 100000000
		// log.Println("新的连接seq:", seq)
		go handleConnection(seq, conn)
	}
}

//读文件
/*func ReadRom(path string) []byte {
	fd, err := ioutil.ReadFile(path)
	if err != nil {
		// log.Println("read file:", path, " error:", err.Error())
		return make([]byte, 0)
	}
	return fd
}
*/
//func NewSendPack(data []byte) []byte {
//	var nlen int16 = int16(len(data))

//	sendData := []byte{0xff, 0xfe, 0xff, 0xfe}
//	sendData = appendInt(sendData, nlen)
//	sendData = appendBytes(sendData, data)
//	sendData = appendBytes(sendData, crcTable(sendData))
//	return sendData
//}

//func uint16ToBytes(d uint16) []byte {
//	buf := bytes.NewBuffer([]byte{})
//	binary.Write(buf, binary.BigEndian, d)
//	return buf.Bytes()
//}

//	{
//		//第一次连接上来，先进行验证
//		key := util.RandomAscii(32)
//		log.Printf("key:%x\n", []byte(key))
//		keyEvent := []byte{0x00, 0x01}
//		keyData := appendBytes(keyEvent, []byte(key))
//		keyPack := NewSendPack(keyData)
//		log.Printf("keyPack sendDat:%x\n", keyPack)
//		conn.Write(keyPack)

//		buf := make([]byte, 65535+8)
//		nLen, err := conn.Read(buf)
//		if err != nil {
//			fmt.Println("Error reading:", err.Error())
//			return
//		}
//		var nextData []byte
//		nextData, pack, err := BuildRevPack(nextData, buf[:nLen])
//		if err != nil {
//			msg := Msg{event_rev, seq, macAddr, conn, pack.content}
//			revChan <- msg
//		}
//		log.Println("read nLen:", nLen)
//	}
