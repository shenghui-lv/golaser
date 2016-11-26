package server

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
)

//func Packet(message []byte) []byte {
//	app
//}

var mock bool = false
var HeaderFieldBytes = []byte{0xff, 0xfe, 0xff, 0xfe}

const (
	ConstHeaderField       = 0xfffefffe
	ConstHeaderLength      = 4
	ConstLengthFieldLength = 2
	ConstCrcFieldLength    = 2
)

func Unpack(buffer []byte) (pack []byte, remain []byte, err error) {

	defer func() {
		if x := recover(); x != nil {
			log.Println("panic:", x)
		}
	}()
	// log.Printf("数据包:%x\n", buffer)
	length := len(buffer)
	// log.Println("unpack buffer len:", length)
	minLength := ConstHeaderLength + ConstLengthFieldLength + ConstCrcFieldLength
	preLength := ConstHeaderLength + ConstLengthFieldLength
	// log.Println("minLen:", minLength, ",preLen:", preLength, ",ConstHeaderField:", ConstHeaderField)
	var i int
	for i = 0; i < length; i++ {
		if length < i+minLength {
			// log.Println("获取的数据包比头字段+长度字段+crc还小")
			break
		}
		if BytesToUint32(buffer[i:i+ConstHeaderLength]) == ConstHeaderField {
			// log.Println("i+ConstHeaderLength:", i+ConstHeaderLength, ",i+ConstLengthFieldLength:", i+ConstLengthFieldLength)
			lenField := buffer[i+ConstHeaderLength : i+preLength]
			contentLength := int(BytesToUint16(lenField))
			// log.Println("contentLength:", contentLength)
			if length < i+minLength+contentLength {
				// log.Println("已经获取到contentLength，但数据包长度不够")
				break
			}
			data := buffer[i+preLength : i+preLength+contentLength]
			crc := buffer[i+preLength+contentLength : i+preLength+contentLength+ConstCrcFieldLength]

			i += minLength + contentLength - 1

			crcData := append(append(HeaderFieldBytes, lenField...), data...)
			rawCrc := CrcTable(crcData)

			//CRC验证

			if BytesToUint16(crc) != BytesToUint16(rawCrc) {
				// log.Printf("CRC校验不合法data:%x, crc:%x, rawCrc:%x\n", data, crc, rawCrc)
				err = errors.New("crc校验失败")
				// log.Println("after new error in crc==========")
				if !mock {
					break
				}
			}
			pack = data
		}
	}
	if mock {

		err = nil
	}
	if i == length {
		remain = make([]byte, 0)
		return
	}
	remain = buffer[i:]

	return
}

//字节转换成有符号整形

/*func BytesToInt16(b []byte) int16 {
	// log.Printf("BytesToInt16 b:%x\n", b)
	bytesBuffer := bytes.NewBuffer(b)

	var x int16
	binary.Read(bytesBuffer, binary.BigEndian, &x)
	// log.Println("BytesToInt16：", x)
	return x
}

func BytesToInt32(b []byte) int32 {
	// log.Printf("BytesToInt32 b:%x\n", b)
	bytesBuffer := bytes.NewBuffer(b)

	var x int32
	binary.Read(bytesBuffer, binary.BigEndian, &x)
	// log.Println("BytesToInt32：", x)
	return x
}*/

//字节转换成无符号整形
func BytesToUint16(b []byte) uint16 {
	// log.Printf("BytesToUint16 b:%x\n", b)
	bytesBuffer := bytes.NewBuffer(b)

	var x uint16
	binary.Read(bytesBuffer, binary.BigEndian, &x)
	// log.Println("BytesToUint16：", x)
	return x
}

func BytesToUint32(b []byte) uint32 {
	// log.Printf("BytesToUint32 b:%x\n", b)
	bytesBuffer := bytes.NewBuffer(b)

	var x uint32
	binary.Read(bytesBuffer, binary.BigEndian, &x)
	// log.Println("BytesToUint32：", x)
	return x
}

func uint16ToBytes(d uint16) []byte {
	buf := bytes.NewBuffer([]byte{})
	binary.Write(buf, binary.BigEndian, d)
	return buf.Bytes()
}

func CrcTable(data []byte) []byte {
	table := []uint16{
		0x0000, 0xC0C1, 0xC181, 0x0140, 0xC301, 0x03C0, 0x0280, 0xC241,
		0xC601, 0x06C0, 0x0780, 0xC741, 0x0500, 0xC5C1, 0xC481, 0x0440,
		0xCC01, 0x0CC0, 0x0D80, 0xCD41, 0x0F00, 0xCFC1, 0xCE81, 0x0E40,
		0x0A00, 0xCAC1, 0xCB81, 0x0B40, 0xC901, 0x09C0, 0x0880, 0xC841,
		0xD801, 0x18C0, 0x1980, 0xD941, 0x1B00, 0xDBC1, 0xDA81, 0x1A40,
		0x1E00, 0xDEC1, 0xDF81, 0x1F40, 0xDD01, 0x1DC0, 0x1C80, 0xDC41,
		0x1400, 0xD4C1, 0xD581, 0x1540, 0xD701, 0x17C0, 0x1680, 0xD641,
		0xD201, 0x12C0, 0x1380, 0xD341, 0x1100, 0xD1C1, 0xD081, 0x1040,
		0xF001, 0x30C0, 0x3180, 0xF141, 0x3300, 0xF3C1, 0xF281, 0x3240,
		0x3600, 0xF6C1, 0xF781, 0x3740, 0xF501, 0x35C0, 0x3480, 0xF441,
		0x3C00, 0xFCC1, 0xFD81, 0x3D40, 0xFF01, 0x3FC0, 0x3E80, 0xFE41,
		0xFA01, 0x3AC0, 0x3B80, 0xFB41, 0x3900, 0xF9C1, 0xF881, 0x3840,
		0x2800, 0xE8C1, 0xE981, 0x2940, 0xEB01, 0x2BC0, 0x2A80, 0xEA41,
		0xEE01, 0x2EC0, 0x2F80, 0xEF41, 0x2D00, 0xEDC1, 0xEC81, 0x2C40,
		0xE401, 0x24C0, 0x2580, 0xE541, 0x2700, 0xE7C1, 0xE681, 0x2640,
		0x2200, 0xE2C1, 0xE381, 0x2340, 0xE101, 0x21C0, 0x2080, 0xE041,
		0xA001, 0x60C0, 0x6180, 0xA141, 0x6300, 0xA3C1, 0xA281, 0x6240,
		0x6600, 0xA6C1, 0xA781, 0x6740, 0xA501, 0x65C0, 0x6480, 0xA441,
		0x6C00, 0xACC1, 0xAD81, 0x6D40, 0xAF01, 0x6FC0, 0x6E80, 0xAE41,
		0xAA01, 0x6AC0, 0x6B80, 0xAB41, 0x6900, 0xA9C1, 0xA881, 0x6840,
		0x7800, 0xB8C1, 0xB981, 0x7940, 0xBB01, 0x7BC0, 0x7A80, 0xBA41,
		0xBE01, 0x7EC0, 0x7F80, 0xBF41, 0x7D00, 0xBDC1, 0xBC81, 0x7C40,
		0xB401, 0x74C0, 0x7580, 0xB541, 0x7700, 0xB7C1, 0xB681, 0x7640,
		0x7200, 0xB2C1, 0xB381, 0x7340, 0xB101, 0x71C0, 0x7080, 0xB041,
		0x5000, 0x90C1, 0x9181, 0x5140, 0x9301, 0x53C0, 0x5280, 0x9241,
		0x9601, 0x56C0, 0x5780, 0x9741, 0x5500, 0x95C1, 0x9481, 0x5440,
		0x9C01, 0x5CC0, 0x5D80, 0x9D41, 0x5F00, 0x9FC1, 0x9E81, 0x5E40,
		0x5A00, 0x9AC1, 0x9B81, 0x5B40, 0x9901, 0x59C0, 0x5880, 0x9841,
		0x8801, 0x48C0, 0x4980, 0x8941, 0x4B00, 0x8BC1, 0x8A81, 0x4A40,
		0x4E00, 0x8EC1, 0x8F81, 0x4F40, 0x8D01, 0x4DC0, 0x4C80, 0x8C41,
		0x4400, 0x84C1, 0x8581, 0x4540, 0x8701, 0x47C0, 0x4680, 0x8641,
		0x8201, 0x42C0, 0x4380, 0x8341, 0x4100, 0x81C1, 0x8081, 0x4040,
	}
	var crc uint16 = 0x0000
	for _, b := range data {
		crc = (crc >> 8) ^ table[(crc^uint16(b))&0x00ff]
	}
	return uint16ToBytes(crc)
}

//0.设备请求要求验证
func isHelloMsg(data []byte) bool {
	nLen := len(data)
	if nLen != 2 {
		return false
	}
	//指令校验
	if BytesToUint16(data) != uint16(1) {
		// log.Printf("指令有误:%x\n", data)
		return false
	}
	return true
}

//1.发送验证key
func SendKey(msg Msg, key string) {
	data := append([]byte{0x00, 0x01}, []byte(key)...)
	nLen := len(data)
	nLenByte := uint16ToBytes(uint16(nLen))
	msg.data = append(msg.data, HeaderFieldBytes...)
	msg.data = append(msg.data, nLenByte...)
	msg.data = append(msg.data, data...)
	crc := CrcTable(msg.data)
	msg.data = append(msg.data, crc...)
	msg.event = event_sendKey
	sendChan <- msg
}

//2.验证收到的checkSum
func checkAndFetchMac(key string, data []byte) (leMac string, leVer string, ret bool) {
	// log.Printf("checkSum验证:%x\n", data)
	nLen := len(data)
	//checksum数据长度 29
	if nLen != 29 {
		log.Println("数据长度有误:", nLen)
		ret = false
		return
	}
	//指令0x00 0x01
	eventCode := data[:2]

	if BytesToUint16(eventCode) != uint16(1) {
		log.Printf("指令有误:%x\n", eventCode)
		ret = false
		return
	}
	sliMac := data[2:8]

	leMac = fmt.Sprintf("%x", sliMac)
	// log.Println("mac:", leMac)

	sliVer := data[8:10]
	//	leVer = fmt.Sprintf("%x", sliVer)
	//	leVer = ByteToHexString(sliVer)
	leVer = string(sliVer)
	// log.Println("ver:", leVer)

	// sliCheck := data[9:28]
	// szCheck := fmt.Sprintf("sliCheck:%x", sliCheck)
	// log.Println("check:", szCheck)
	//TODO  进行校验
	// check according to protocol
	ret = true
	return
}

//取奇数字符组成字符串
func Odd(szOrial string) string {
	nlen := len(szOrial)
	sliOdd := make([]byte, (nlen>>2)+1)
	i := 0
	for index, c := range szOrial {
		if (index+1)%2 != 0 {
			sliOdd[i] = byte(c)
			i++
		}
	}
	return string(sliOdd)
}

//3.1验证成功通知设备
func SendWelcome(msg Msg) {
	msg.event = event_send
	data := []byte{0xff, 0xfe, 0xff, 0xfe, 0x00, 0x03, 0x00, 0x01, 0x01}
	crc := CrcTable(data)
	msg.data = append(data, crc...)
	sendChan <- msg
}

//3.2验证失败通知设备
func SendFail(msg Msg) {
	msg.event = event_send
	data := []byte{0xff, 0xfe, 0xff, 0xfe, 0x00, 0x03, 0x00, 0x01, 0x00}
	crc := CrcTable(data)
	msg.data = append(data, crc...)
	sendChan <- msg
}

func SendCalibration(msg Msg, d []byte) {
	msg.event = event_send
	nLen := len(d)
	//添加2字节的指令字段长度
	nLen += 2
	data := []byte{0xff, 0xfe, 0xff, 0xfe}
	data = append(data, uint16ToBytes(uint16(nLen))...)
	data = append(data, []byte{0x00, 0x03}...)
	data = append(data, d...)
	crc := CrcTable(data)
	msg.data = append(data, crc...)
	sendChan <- msg
}

func SendOTAData(msg Msg, d []byte, ver string, nTotal uint16, nIndex uint16) {
	msg.event = event_send
	data := []byte{0xff, 0xfe, 0xff, 0xfe, 0x04, 0x08, 0x00, 0x05}
	data = append(data, []byte(ver)...)
	data = append(data, uint16ToBytes(nTotal)...)
	data = append(data, uint16ToBytes(nIndex)...)
	data = append(data, d...)
	data = append(data, CrcTable(data)...)
	msg.data = data
	sendChan <- msg
}

//发送关闭连接消息
func SendClose(msg Msg) {
	msg.event = event_close
	sendChan <- msg
}

/*
func SendOTAStar(msg Msg, ver string, nNumPack uint16) {
	msg.event = event_send
	data := []byte{0xff, 0xfe, 0xff, 0xfe, 0x00, 0x06, 0x00, 0x05}
	data = append(data, []byte(ver)...)
	data = append(data, uint16ToBytes(nNumPack)...)
	data = append(data, CrcTable(data)...)
	msg.data = data
	go func(themsg Msg) {
		// log.Println("OTA begin sleep.....")
		time.Sleep(ota_timeout - 10e9)
		sendChan <- themsg
		// log.Println("OTA wake up and send to sendChan....")
	}(msg)

}
*/

//获取内容的指令字段
func GetAction(d []byte) (ret []byte, err error) {
	nLen := len(d)
	if nLen < 2 {
		err = errors.New("长度不对")
		return
	}
	ret = d[:2]
	return
}

//保存数据库
func DoAction2(msg Msg) {

//	SaveDataDB(msg)
}

//OTA
func DoAction5(msg Msg) {

}

func ByteToHexString(data []byte) string {

	hex := []byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f'}
	nLen := len(data)
	buff := make([]byte, 2*nLen)
	for i := 0; i < nLen; i++ {
		buff[2*i] = hex[(data[i]>>4)&0x0f]
		buff[2*i+1] = hex[data[i]&0x0f]
	}
	szHex := string(buff)

	return szHex

}

func HexStringToByte(hex string) []byte {
	len := len(hex) / 2
	result := make([]byte, len)
	i := 0
	for i = 0; i < len; i++ {
		pos := i * 2
		result[i] = ToByte(hex[pos])<<4 | ToByte(hex[pos+1])
	}
	return result
}
func ToByte(c uint8) byte {

	if c >= '0' && c < '9' {
		return byte(c - '0')
	}
	if c >= 'a' && c <= 'z' {
		return byte(c - 'a' + 10)
	}
	if c >= 'A' && c <= 'Z' {
		return byte(c - 'A' + 10)
	}
	return 0
}
