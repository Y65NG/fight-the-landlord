package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func main() {
	// conn, err := net.Dial("tcp", "45.77.149.81:8888")
	conn, err := net.Dial("tcp", "127.0.0.1:8888")
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()
	go sendData(conn)
	for {
		buf := [10240]byte{}
		n, err := conn.Read(buf[:])
		if err != nil {
			return
		}
		fmt.Print(string(buf[:n]))
	}
}

func sendData(conn net.Conn) {
	reader := bufio.NewReader(os.Stdin)
	for {
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		_, err := conn.Write([]byte(line + "\n"))
		if err != nil {
			log.Fatalln(err)
		}
	}
}
