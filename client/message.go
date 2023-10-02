package main

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

// type client struct {
// 	Nick string   `json:"nick"`
// 	Conn net.Conn `json:"conn"`
// }

type message struct {
	MsgType messageType `json:"msg_type"`
	Content string      `json:"content"`
	Sender  string      `json:"sender"`
}
