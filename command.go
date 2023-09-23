package main

type commandID int

const (
	CMD_LIST_COMMANDS commandID = iota
	CMD_QUIT
	CMD_READY
	CMD_VIEW_CARDS
	CMD_USE_CARDS
	CMD_PASS
)

type command struct {
	id     commandID
	sender *client
	args   []string
}
