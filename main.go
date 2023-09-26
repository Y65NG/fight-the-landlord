package main

import (
	"log"
	"net"
)

func main() {
	listen, err := net.Listen("tcp", "0.0.0.0:8888")
	if err != nil {
		log.Fatalf("unable to start server: %s", err.Error())
	}
	defer listen.Close()
	log.Println("server started on :8888")

	server := newServer()
	go server.runCommands()
	go server.gameLoop()
	// go server.removeClosedClient()

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Printf("unable to accept connection: %s", err.Error())
		}
		log.Printf("client has connected: %s", conn.RemoteAddr().String())
		go server.newClient(conn)
	}
}
