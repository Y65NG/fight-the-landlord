package main

import (
	"bufio"
	"encoding/json"
	"log"
	"net"
	"strings"
	"time"
)

type client struct {
	Nick     string `json:"nick"`
	commands chan<- command
	Conn     net.Conn `json:"conn"`
}

func (c *client) readInput() {
	reader := bufio.NewReader(c.Conn)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		msg = strings.Trim(msg, "\r\n ")
		log.Printf("%v (%v) -> %v", c.Nick, c.Conn.RemoteAddr(), msg)
		if len(msg) > 0 && msg[0] != '/' {
			c.commands <- command{CMD_MESSAGE, c, []string{msg}}
			continue
		}
		args := strings.Split(msg, " ")
		cmd := strings.TrimSpace(args[0])

		switch cmd {
		case "":
			c.commands <- command{CMD_EMPTY_LINE, c, args}
		case "/commands":
			c.commands <- command{CMD_LIST_COMMANDS, c, args}
		case "/list":
			c.commands <- command{CMD_LIST_PLAYERS, c, args}
		case "/quit":
			c.commands <- command{CMD_QUIT, c, args}
		case "/ready":
			c.commands <- command{CMD_READY, c, args}
		case "/view":
			c.commands <- command{CMD_VIEW_CARDS, c, args}
		case "/use":
			c.commands <- command{CMD_USE_CARDS, c, args}
		case "/pass":
			c.commands <- command{CMD_PASS, c, args}
		default:
			c.commands <- command{CMD_UNKNOWN, c, args}

		}
	}
}

type messageType int

const (
	MSG_MESSAGE messageType = iota
	MSG_ERROR
	MSG_PLAYER_STATUS
	MSG_INFO
	MSG_CHAT
	MSG_ROOM_INFO
	MSG_STOP
)

type Message struct {
	MsgType messageType `json:"msg_type"`
	Content string      `json:"content"`
	Sender  string      `json:"sender"`
}

func (c *client) msg(msgType messageType, msg string) {
	byts, err := json.Marshal(Message{msgType, msg, c.Nick})
	if err != nil {
		log.Println(err)
		return
	}
	c.Conn.Write([]byte(string(byts) + "\n"))
	time.Sleep(300 * time.Millisecond)
	log.Printf("%v (%v) <- %v", c.Nick, c.Conn.RemoteAddr(), strings.Trim(msg, "\r\n\b "))
}

// func (c *client) prompt() {
// 	c.Conn.Write([]byte("> "))
// }

func (c *client) err(e error) {

	byts, err := json.Marshal(Message{MSG_INFO, e.Error(), c.Nick})
	if err != nil {
		log.Println(err)
		return
	}
	c.Conn.Write([]byte(string(byts) + "\n"))
	log.Printf("%v (%v) <- %v", c.Nick, c.Conn.RemoteAddr(), strings.Trim(e.Error(), "\r\n\b "))
}
