package main

import (
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

func main() {
	listen, err := net.Listen("tcp", "0.0.0.0:8888")
	if err != nil {
		log.Fatalf("unable to start server: %s", err.Error())
	}
	defer listen.Close()
	log.Println("server started on :8888")

	server := newServer()

	args := os.Args[1:]
	if len(args) > 0 {
		if n, err := strconv.Atoi(args[0]); err == nil {
			server.SetNumPlayers(n)
		}
	}

	go server.runCommands()
	go server.gameLoop()
	go server.removeClosedClient()

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Printf("unable to accept connection: %s", err.Error())
		}
		conn.SetDeadline(time.Now().Add(120 * time.Second))
		if err != nil {
			log.Printf("unable to set keep alive: %s", err.Error())
		}

		log.Printf("client has connected: %s", conn.RemoteAddr().String())
		go server.newClient(conn)
	}
}
