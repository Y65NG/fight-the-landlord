package main

import (
	"log"
	"net"
	"os"
	"strconv"
	"time"
	"landlord/server"
)

func main() {
	f, err := os.OpenFile("server.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		log.Fatalf("unable to open log file: %s", err.Error())
	}
	defer f.Close()
	log.SetOutput(f)
	log.Println("______________________")
	listen, err := net.Listen("tcp", "0.0.0.0:8888")
	if err != nil {
		log.Fatalf("unable to start server: %s", err.Error())
	}
	defer listen.Close()
	log.Println("server started on port 8888")

	server := server.NewServer()

	args := os.Args[1:]
	if len(args) > 0 {
		if n, err := strconv.Atoi(args[0]); err == nil {
			server.SetNumPlayers(n)
		}
	}

	go server.RunCommands()
	go server.GameLoop()
	go server.RemoveClosedClient()

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Printf("unable to accept connection: %s", err.Error())
		}
		conn.SetDeadline(time.Now().Add(300 * time.Second))
		if err != nil {
			log.Printf("unable to set keep alive: %s", err.Error())
		}

		log.Printf("client has connected: %s", conn.RemoteAddr().String())
		go server.NewClient(conn)
	}
}
