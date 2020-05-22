package main

import (
	"github.com/grandcat/zeroconf"
	"google.golang.org/grpc"
	"log"
	"net"
)






type Target []struct {
	Name            string   `json:"name"`
	ProcessName     string   `json:"processName"`
	TriggerCommands []string `json:"triggerCommands"`
	Type            string   `json:"type"`
}



func init() {
	readKeyCodeFile()
	readMonitorFile()
}


func main() {

	//init keyevent
	go getKeyEvent()

	// triggering monitoring
	if targetProc != nil && len(targetProc) > 0{
		for _, s := range targetProc {
			go serviceMonitor(s)
		}
	}else {
		log.Println("Monitor File not found.")
	}

	//MDNS PROCESS
	nets, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	mdnsServer, err := zeroconf.Register(
		"CloudTv_"+getEmac(),
		"_http._tcp",
		"local.",
		50051,
		getTvData(),
		nets)

	if err != nil {
		panic(err)
	}

	defer mdnsServer.Shutdown()

	//GRPC PROCESS
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		panic("error while init server")
	}

	server := grpc.NewServer()
	defer server.GracefulStop()
	RegisterTvInteractionServiceServer(server, Server{})
	if err := server.Serve(lis); err != nil {
		panic(err)
	}
}











































