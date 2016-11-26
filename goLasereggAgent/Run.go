package main

import (
	"flag"
	"fmt"
	"goLasereggAgent/server"
	"log"
	"os"
	"runtime"
	"time"
	//	"github.com/xladykiller/gotest/server"
)

func main() {
	//	runtime.GOMAXPROCS(runtime.NumCPU() * 2)   this is tricky
	runtime.GOMAXPROCS(runtime.NumCPU())
	/*	arg_num := len(os.Args)
		var port string
		if arg_num<3 || os.Args[1] != "-p" {
			fmt.Println("Please specify port\n>goLasereggAgent -p 21601")
			return
		}	else {
			port = os.Args[2]
		}*/
	initLog()
	port := "18000" // 8000 for v1, 18000 for FieldEgg, 21601 for V16.01
	fmt.Println("Main thread start, listening to port ( " + port + " )........................")
	server.Server(port)
}

var (
	logFileName = flag.String("log", "gateway_"+time.Now().Format("2006-01-02")+".log", "Log file name")
)

func initLog() {
	flag.Parse()
	logFile, err := os.OpenFile(*logFileName, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("Fail to find", *logFile, "server start Failed")
		os.Exit(1)
	}
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime)
	log.Println("=============================================")
}
