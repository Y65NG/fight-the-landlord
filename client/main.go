package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

var msgChan = make(chan message, 10)
var sendChan = make(chan string, 10)

func main() {
	f, err := os.OpenFile("server.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		log.Fatalf("unable to open log file: %s", err.Error())
	}
	defer f.Close()
	log.SetOutput(f)
	log.Println("______________________")
	args := os.Args[1:]
	var conn net.Conn

	if len(args) == 0 {
		conn, err = net.Dial("tcp", "45.77.149.81:8888")
		// conn, err = net.Dial("tcp", "127.0.0.1:8888")
	} else {
		conn, err = net.Dial("tcp", args[0])
	}

	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()
	done := make(chan struct{})
	setName(conn, done)
	<-done
	log.Println("Done")
	go sendData(conn)
	go listenData(conn)
	time.Sleep(100 * time.Millisecond)
	Run()

}

func setName(conn net.Conn, done chan struct{}) {
	finished := false
	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			if finished {
				return
			} else {

				line, err := reader.ReadString('\n')
				if err != nil {
					panic(err)
				}
				conn.Write([]byte(line + "\n"))
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
	go func() {
		reader := bufio.NewReader(conn)
		fmt.Print("Please type your nickname: ")
		for {
			line, err := reader.ReadString('\n')
			log.Println(line)
			if err != nil {
				panic(err)
			}
			if line == "ok\n" {
				done <- struct{}{}
				finished = true
				break
			} else {
				fmt.Print("Please provide a valid nickname: ")
			}
		}
		return
	}()
	return
}

func sendData(conn net.Conn) {
	for {
		line := <-sendChan
		_, err := conn.Write([]byte(line + "\n"))
		if err != nil {
			// log.Println(err)
			panic(err)
		}

	}
}

func listenData(conn net.Conn) {
	for {
		buf := [10240]byte{}

		n, err := conn.Read(buf[:])

		if err != nil {
			return
		}
		var msg message
		dec := json.NewDecoder(strings.NewReader(string(buf[:n])))
		for {
			if err := dec.Decode(&msg); err != nil {
				break
			}
			msgChan <- msg
		}
		// err = json.Unmarshal(buf[:n], &msg)
		// if err != nil {
			// log.Println(string(buf[:n]))	
			// panic(err)
		// }
		// log.Println(msg)
		// msgChan <- msg
		// fmt.Print(string(buf[:n]))
	}
}
