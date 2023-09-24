package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/TwiN/go-color"
)

type client struct {
	nick     string
	commands chan<- command
	conn     net.Conn
}

func (c *client) readInput() {
	reader := bufio.NewReader(c.conn)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		msg = strings.Trim(msg, "\r\n ")
		args := strings.Split(msg, " ")
		cmd := strings.TrimSpace(args[0])

		switch cmd {
		case "":
			c.commands <- command{CMD_EMPTY_LINE, c, args}
		case "/commands":
			c.commands <- command{CMD_LIST_COMMANDS, c, args}
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
			c.err(errors.New("unknown command. Type /commands to see available commands"))
		}
	}
}

func (c *client) msg(msg string) {
	c.conn.Write([]byte(color.With(color.Gray, fmt.Sprintf("%s\n", msg))))
}

func (c *client) prompt() {
	c.conn.Write([]byte(color.With(color.Bold, "> ")))
}

func (c *client) err(err error) {
	c.conn.Write([]byte(color.With(color.Red, fmt.Sprintf("Err: %s\n", err.Error()))))
}
