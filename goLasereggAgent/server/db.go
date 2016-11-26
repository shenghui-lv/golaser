package server

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"strings"
)

var db *sql.DB
var laseregg_database *sql.DB

/*func InitDb() bool {
	var err error
	if db, err = sql.Open("mysql", "root:9ce9023d80@tcp(123.57.1.4:3306)/top?charset=utf8"); err != nil {
		log.Println("sql.Open err:", err.Error())
		db = nil
		return false
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	return true
}*/

func InitLasereggDatabase() bool {
	var err error
	if laseregg_database, err = sql.Open("mysql", "golasereggagent:psw_tnegaggeresalog@tcp(rds24m33zw0xmn687nt8.mysql.rds.aliyuncs.com:3306)/laseregg?charset=utf8"); err != nil {
		log.Println("Cannot open laseregg_database", err.Error())
		laseregg_database = nil
		return false
	}
	laseregg_database.SetMaxOpenConns(10)
	laseregg_database.SetMaxIdleConns(5)
	return true
}

func InitLasereggDatabaseFromAWS() bool {
	var err error
	if laseregg_database, err = sql.Open("mysql", "golasereggagent:psw_tnegaggeresalog@tcp(laseregg.mysql.rds.aliyuncs.com:3306)/laseregg?charset=utf8"); err != nil {
		log.Println("Cannot open laseregg_database from aws", err.Error())
		laseregg_database = nil
		return false
	}
	laseregg_database.SetMaxOpenConns(10)
	laseregg_database.SetMaxIdleConns(5)
	return true
}

// for updating handshake time
func HandshakeDB(msg Msg) {

	sqlTime := "select lastHandshakeTime,firstHandshakeTime,version from top where mac=?"
	stmt_handshake, err := db.Prepare(sqlTime)
	defer stmt_handshake.Close()
	if err != nil {
		log.Println("[DB]", err.Error())
		return
	}

	rows, err := stmt_handshake.Query(msg.macAddr)
	defer rows.Close()
	if err != nil {
		log.Println("[DB]", err.Error())
		return
	}

	var lastHandshakeTime string
	var firstHandshakeTime string
	var version string

	leVer := fmt.Sprintf("%02d.%02d", msg.data[8], msg.data[9])

	for rows.Next() {
		err = rows.Scan(&lastHandshakeTime, &firstHandshakeTime, &version)
		//log.Println("lastHandshakeTime=<", lastHandshakeTime, "> firstHandshakeTime=<", firstHandshakeTime, ">")
		if firstHandshakeTime == "" || firstHandshakeTime == "0000-00-00 00:00:00" {
			stmt_update, err := db.Prepare("update top set `firstHandshakeTime`=now(),`lastHandshakeTime`=now() where mac=?")
			defer stmt_update.Close()
			if err != nil {
				log.Println("[DB]", err.Error())
				return
			}

			if _, err := stmt_update.Exec(msg.macAddr); err != nil {
				log.Println("[DB]", err.Error())
			}

		} else {
			stmt_update, err := db.Prepare("update top set `lastHandshakeTime`=now() where mac=?")
			defer stmt_update.Close()
			if err != nil {
				log.Println("[DB]", err.Error())
				return
			}
			if _, err := stmt_update.Exec(msg.macAddr); err != nil {
				log.Println("[DB]", err.Error())
			}
		}

		if version != leVer {
			stmt_update, err := db.Prepare("update top set version=? where mac=?")
			defer stmt_update.Close()
			if err != nil {
				log.Println("[DB]", err.Error())
				return
			}
			if _, err := stmt_update.Exec(leVer, msg.macAddr); err != nil {
				log.Println("[DB]", err.Error())
			}
		}
	}
}

func SendCaliDB(msg Msg) {
	// log.Println("标定")
	szQueryLocation := "select calibrationCode from top where mac=?"
	stmt1, err := db.Prepare(szQueryLocation)
	defer stmt1.Close()
	if err != nil {
		return
	}
	rows, err := stmt1.Query(msg.macAddr)
	defer rows.Close()
	if err != nil {
		return
	}

	for rows.Next() {
		var calibrationCode string
		err = rows.Scan(&calibrationCode)
		if calibrationCode == "" {
			log.Println("Missing calibrationCode for [", msg.macAddr, "]")
		}
		if err != nil {
			// log.Println("rows.scan err:", err.Error())
		} else {
			var sendMsg Msg
			sendMsg.conn = msg.conn
			sendMsg.macAddr = msg.macAddr
			sendMsg.seq = msg.seq
			SendCalibration(sendMsg, HexStringToByte(calibrationCode))
		}
	}
}

//保存数据到数据库
func SaveDataDB(msg Msg) {
	nLen := len(msg.data)
	// log.Println("RECV:(", nLen, ")", msg.data)
	if nLen < 20 {
		// log.Println("接收到要保存的数据长度小于18")
	}
	szMacAddr := msg.macAddr
	// log.Println("macAddr:", szMacAddr)
	baseIndex := 0

	// log.Printf("before0:%x\n", msg.data)
	// action := msg.data[baseIndex : baseIndex+2]
	// log.Printf("action:%x\n", action)
	// log.Printf("before1:%x\n", msg.data)
	// jump the first two, weil das ist the packet type identifier
	baseIndex += 2
	//------------------------------

	// location := msg.data[baseIndex : baseIndex+2]
	// log.Printf("location:%x\n", location)
	// log.Printf("after:%x->baseIndex:%d\n", msg.data, baseIndex)
	// jump the second two, weil das ist the city

	baseIndex += 2
	//------------------------------

	//原始pm2.5
	opm2_5 := msg.data[baseIndex : baseIndex+2]
	szOpm2_5 := BytesToUint16(opm2_5) //fmt.Sprintf("%x", opm2_5)
	baseIndex += 2
	//------------------------------

	//原始pm10
	opm10 := msg.data[baseIndex : baseIndex+2]
	szOpm10 := BytesToUint16(opm10) //fmt.Sprintf("%x", opm10)
	baseIndex += 2
	//------------------------------

	//标定后pm2.5
	pm2_5 := msg.data[baseIndex : baseIndex+2]
	szPm2_5 := BytesToUint16(pm2_5) //fmt.Sprintf("%x", pm2_5)
	baseIndex += 2
	//------------------------------

	//标定后pm10
	pm10 := msg.data[baseIndex : baseIndex+2]
	szPm10 := BytesToUint16(pm10) //fmt.Sprintf("%x", pm10)
	baseIndex += 2
	//------------------------------

	//pm2.5颗粒数
	dust2_5 := msg.data[baseIndex : baseIndex+2]
	szDust2_5 := BytesToUint16(dust2_5) //fmt.Sprintf("%x", dust2_5)
	baseIndex += 2
	//------------------------------

	//pm0.3颗粒数
	dust0_3 := msg.data[baseIndex : baseIndex+2]
	szDust0_3 := BytesToUint16(dust0_3) //fmt.Sprintf("%x", dust0_3)
	baseIndex += 2
	//------------------------------

	//湿度
	humidity := msg.data[baseIndex : baseIndex+2]
	szHumidty := BytesToUint16(humidity)
	//fmt.Sprintf("humidity:%x", humidity)
	// log.Printf("humidity:%x", humidity)
	baseIndex += 2
	//------------------------------

	//温度
	szTemp8 := int8(msg.data[baseIndex+1])
	var szTemp int
	//szTemp := BytesToInt16(temp) //fmt.Sprintf("%x", temp)
	//temp := msg.data[baseIndex : baseIndex+2]
	//szTemp := BytesToInt16(temp) //fmt.Sprintf("%x", temp)
	//newtemp = (1.2398 * temp) - 12.731
	if szHumidty != 0 {
		szTemp = int((float64(szTemp8) * 1.2398) - 12.731)
	}
	baseIndex += 2
	//------------------------------

	//=================================
	// do DB action, for capatible reason
	szSql := "insert into `topdata` (`recieveTime`,`temperature`,`humidity`,`pm2_5`,`mac`,`pm2_5_count`,`pm10_count`,`pm10`,`c_pm2_5`,`c_pm_10`) values (now(),?,?,?,?,?,?,?,?,?)"
	stmt, err := db.Prepare(szSql)
	defer stmt.Close()
	if err != nil {
		log.Println("db.Prepare err:", err.Error())
		return
	}

	_, err = stmt.Exec( /*time.Now().String(),*/ szTemp, szHumidty, szOpm2_5, szMacAddr, szDust2_5, szDust0_3, szOpm10, szPm2_5, szPm10)
	//res, err
	if err != nil {
		log.Println("stmt.Exec err:", err.Error())
		return
	}
	// id, err := res.LastInsertId()
	// log.Println("the last insert id is ", id)
	//=================================

	/* not used in current version, but for next version

	// device  & sensor running time
	if baseIndex + 4 <= nLen {
		deviceTime := msg.data[baseIndex : baseIndex+2]
		leDeviceTime := BytesToUint16(deviceTime) / 2 // hours
		sensorTime := msg.data[baseIndex+2 : baseIndex+4]
		leSensorTime := BytesToUint16(sensorTime) / 2 //  hours
	}
	baseIndex += 4
	//------------------------------

	//=================================
	// do DB action
	szSql := "insert into `topdata` (`recieveTime`,`temperature`,`humidity`,`pm2_5`,`mac`,`pm2_5_count`,`pm10_count`,`pm10`,`c_pm2_5`,`c_pm_10`,`deviceUsed`,`sensorUsed`) values (now(),?,?,?,?,?,?,?,?,?,?,?)"
	stmt, err := db.Prepare(szSql)
	defer stmt.Close()
	if err != nil {
		// log.Println("db.Prepare err:", err.Error())
		return
	}

	_, err := stmt.Exec(szTemp, szHumidty, szOpm2_5, szMacAddr, szDust2_5, szDust0_3, szOpm10, szPm2_5, szPm10,deviceTime,sensorTime)
	//res, err := stmt.Exec
	// id, err := res.LastInsertId()
	if err != nil {
		// log.Println("res.LastInsertId err:", err.Error())
		return
	}
	// log.Println("the last insert id is ", id)
	//=================================
	*/
}

// sychronize data to new database

func Handshake_LE_DB(msg Msg) {

	leVer := fmt.Sprintf("%02d.%02d", msg.data[8], msg.data[9])
	conn := strings.Split(msg.conn.RemoteAddr().String(), ":")

	// sqlHandshake := "call `laseregg`.`handshake`(?,?,?,?)"
	sqlHandshake := fmt.Sprintf("call `laseregg`.`handshake`('%s','%s','%s','%s')",
		msg.macAddr, leVer, conn[0], conn[1])
	log.Println(sqlHandshake)
	stmt_handshake, err := laseregg_database.Prepare(sqlHandshake)
	defer stmt_handshake.Close()
	if err != nil {
		log.Println("[LE DB handshake prepare] ", err.Error())
		return
	}

	//	arg1 := fmt.Sprintf("%s", msg.macAddr)
	//	arg2 := fmt.Sprintf("%s", leVer)
	//	arg3 := fmt.Sprintf("%s", conn[0])
	//	arg4 := fmt.Sprintf("%s", conn[1])
	//	log.Println(arg1, arg2, arg3, arg4)

	if _, err = stmt_handshake.Exec(); err != nil {
		log.Println("[LE DB handshake exec] ", err.Error())
		return
	}
}

func SendCali_LE_DB(msg Msg) {
	log.Println("SendCali_LE_DB")
	szQueryLocation := "select calibrationCode from `laseregg`.`laseregg` where mac=?"
	stmt1, err := laseregg_database.Prepare(szQueryLocation)
	defer stmt1.Close()
	if err != nil {
		log.Println("[LE DB sendcali prepare] ", err.Error())
		return
	}
	rows, err := stmt1.Query(msg.macAddr)
	defer rows.Close()
	if err != nil {
		log.Println("[LE DB sendcali query] ", err.Error())
		return
	}

	for rows.Next() {
		var calibrationCode string
		err = rows.Scan(&calibrationCode)
		if calibrationCode == "" {
			log.Println("Missing calibrationCode for [", msg.macAddr, "]")
		}
		if err != nil {
			// log.Println("rows.scan err:", err.Error())
		} else {
			var sendMsg Msg
			sendMsg.conn = msg.conn
			sendMsg.macAddr = msg.macAddr
			sendMsg.seq = msg.seq
			SendCalibration(sendMsg, HexStringToByte(calibrationCode))
		}
	}
}

func SaveData_LE_DB(msg Msg) {
	nLen := len(msg.data)
	// log.Println("RECV:(", nLen, ")", msg.data)
	if nLen < 20 {
		// log.Println("接收到要保存的数据长度小于18")
	}
	// log.Println("macAddr:", szMacAddr)
	baseIndex := 0

	// log.Printf("before0:%x\n", msg.data)
	// action := msg.data[baseIndex : baseIndex+2]
	// log.Printf("action:%x\n", action)
	// log.Printf("before1:%x\n", msg.data)
	// jump the first two, weil das ist the packet type identifier
	baseIndex += 2
	//------------------------------

	// location := msg.data[baseIndex : baseIndex+2]
	// log.Printf("location:%x\n", location)
	// log.Printf("after:%x->baseIndex:%d\n", msg.data, baseIndex)
	// jump the second two, weil das ist the city

	baseIndex += 2
	//------------------------------

	//原始pm2.5
	opm2_5 := msg.data[baseIndex : baseIndex+2]
	szOpm2_5 := BytesToUint16(opm2_5) //fmt.Sprintf("%x", opm2_5)
	baseIndex += 2
	//------------------------------

	//原始pm10
	opm10 := msg.data[baseIndex : baseIndex+2]
	szOpm10 := BytesToUint16(opm10) //fmt.Sprintf("%x", opm10)
	baseIndex += 2
	//------------------------------

	//标定后pm2.5
	pm2_5 := msg.data[baseIndex : baseIndex+2]
	szPm2_5 := BytesToUint16(pm2_5) //fmt.Sprintf("%x", pm2_5)
	baseIndex += 2
	//------------------------------

	//标定后pm10
	pm10 := msg.data[baseIndex : baseIndex+2]
	szPm10 := BytesToUint16(pm10) //fmt.Sprintf("%x", pm10)
	baseIndex += 2
	//------------------------------

	//pm2.5颗粒数
	dust2_5 := msg.data[baseIndex : baseIndex+2]
	szDust2_5 := BytesToUint16(dust2_5) //fmt.Sprintf("%x", dust2_5)
	baseIndex += 2
	//------------------------------

	//pm0.3颗粒数
	dust0_3 := msg.data[baseIndex : baseIndex+2]
	szDust0_3 := BytesToUint16(dust0_3) //fmt.Sprintf("%x", dust0_3)
	baseIndex += 2
	//------------------------------

	//湿度
	humidity := msg.data[baseIndex : baseIndex+2]
	szHumidty := BytesToUint16(humidity)
	//fmt.Sprintf("humidity:%x", humidity)
	// log.Printf("humidity:%x", humidity)
	baseIndex += 2
	//------------------------------

	//温度
	szTemp8 := int8(msg.data[baseIndex+1])
	var szTemp int
	//szTemp := BytesToInt16(temp) //fmt.Sprintf("%x", temp)
	//temp := msg.data[baseIndex : baseIndex+2]
	//szTemp := BytesToInt16(temp) //fmt.Sprintf("%x", temp)
	//newtemp = (1.2398*temp) -12.731
	if msg.macAddr == "d3e818253198" {
		szTemp = int(float64(szTemp8))
	} else if szHumidty != 0 {
		szTemp = int((float64(szTemp8) * 1.2398) - 12.731)
	} else {
		szTemp = 0
	}
	baseIndex += 2
	//------------------------------

	var leDeviceTime uint16 = 0
	var leSensorTime uint16 = 0

	// device  & sensor running time
	if baseIndex+4 <= nLen {
		deviceTime := msg.data[baseIndex : baseIndex+2]
		leDeviceTime = BytesToUint16(deviceTime) / 2 // hours
		sensorTime := msg.data[baseIndex+2 : baseIndex+4]
		leSensorTime = BytesToUint16(sensorTime) / 2 //  hours
	}
	baseIndex += 4
	//------------------------------

	//=================================
	// do DB action
	//	szSql := "call `laseregg`.`upload_le_data`(?,?,?,?,?,?,?,?,?,?,?)"
	sqlUploadData := fmt.Sprintf("call `laseregg`.`upload_le_data`('%s',%d,%d,%d,%d,%d,%d,%d,%d,%d,%d)",
		msg.macAddr, szOpm2_5, szDust2_5, szOpm10, szDust0_3, szPm2_5, szPm10, leDeviceTime, leSensorTime, szTemp, szHumidty)
	log.Println(sqlUploadData)

	stmt, err := laseregg_database.Prepare(sqlUploadData)
	defer stmt.Close()
	if err != nil {
		log.Println("[LE DB upload data prepare] ", err.Error())
		return
	}

	_, err = stmt.Exec()
	//res, err := stmt.Exec
	// id, err := res.LastInsertId()
	if err != nil {
		log.Println("[LE DB upload data query] ", err.Error())
		return
	}
	// log.Println("the last insert id is ", id)
	//=================================
}
