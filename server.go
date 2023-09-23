package main

import (
	"bufio"
	"errors"
	"fmt"
	"landlord/util"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)


type server struct {
	commands chan command
	members  map[net.Addr]*client
	game     *util.Game
}

func newServer() *server {
	return &server{
		commands: make(chan command, 1),
		members:  make(map[net.Addr]*client),
		game:     util.NewGame(),
	}
}

func (s *server) newClient(conn net.Conn) {
	c := &client{
		commands: s.commands,
		conn:     conn,
	}
	s.members[conn.RemoteAddr()] = c
	s.game.AddPlayer(c.conn)
	c.msg("please set your nickname:")
	c.prompt()
	nickname, err := bufio.NewReader(conn).ReadString('\n')
	for nickname == "\n" || err != nil {
		c.err(errors.New("please type a valid name"))
		nickname, err = bufio.NewReader(conn).ReadString('\n')
	}
	nickname = strings.Trim(nickname, "\n")
	c.nick = nickname
	c.msg(fmt.Sprintf("welcome to the server, %s. Type /commands to see available commands", nickname))
	c.prompt()
	go c.readInput()
}

func (s *server) runCommands() {
	for command := range s.commands {
		sender := command.sender
		switch command.id {
		case CMD_LIST_COMMANDS:
			s.listCommands(sender)
		case CMD_QUIT:
			s.quit(sender)
		case CMD_READY:
			s.ready(sender)
		case CMD_VIEW_CARDS:
			s.viewCards(sender, command.args)
		case CMD_USE_CARDS:
			s.useCards(sender, command.args)
		case CMD_PASS:
			s.pass(sender)
		}
	}
}

func (s *server) gameLoop() {
	for {
		switch s.game.State {
		case util.STATE_WAITING:
			if s.game.NumReady() == util.NUM_PLAYERS {
				s.game.NextState()
			}
		case util.STATE_PLAYING:
			time.Sleep(1 * time.Second)
			s.play()
		}
	}
}

func (s *server) play() {
	var players []*util.Player
	g := s.game
	g.Players.Range(func(_, player any) bool {
		if player, ok := player.(*util.Player); ok {
			player.Deal(&g.Deck, 51/util.NUM_PLAYERS)
			player.Sort()
			players = append(players, player)
		}
		return true
	})

	landlordIdx := util.R.Intn(util.NUM_PLAYERS)
	g.Landlord = players[landlordIdx]
	g.Landlord.Deal(&g.Deck, 3)
	g.Landlord.Position = util.LANDLORD

	s.members[g.Landlord.Conn.RemoteAddr()].msg("you are the landlord")
	s.broadcast(s.members[g.Landlord.Conn.RemoteAddr()], fmt.Sprintf("%s is the landlord", s.members[g.Landlord.Conn.RemoteAddr()].nick))

	currentPlayerIdx := landlordIdx
	time.Sleep(1 * time.Second)
	for _, player := range s.members {
		player, _ := g.Players.Load(player.conn.RemoteAddr())
		if player.(*util.Player).Position != util.LANDLORD {
			s.members[player.(*util.Player).Conn.RemoteAddr()].msg("your cards: " + player.(*util.Player).String())
		}
	}
	for {
		g.CurrentPlayer = players[currentPlayerIdx]
		s.members[g.CurrentPlayer.Conn.RemoteAddr()].msg("it's your turn")
		s.viewCards(s.members[g.CurrentPlayer.Conn.RemoteAddr()], []string{})
		s.members[g.CurrentPlayer.Conn.RemoteAddr()].prompt()
		s.broadcast(s.members[g.CurrentPlayer.Conn.RemoteAddr()], fmt.Sprintf("waiting for %s's action...", s.members[g.CurrentPlayer.Conn.RemoteAddr()].nick))

		cards := <-g.CurrentUsedCards
		if len(g.CurrentPlayer.Cards) == 0 {
			s.broadcast(nil, fmt.Sprintf("%s won the game", s.members[g.CurrentPlayer.Conn.RemoteAddr()].nick))
			g.NextState()
			break
		}
		if len(cards) == 0 {
			currentPlayerIdx = (currentPlayerIdx + 1) % util.NUM_PLAYERS
			continue
		}

		currentPlayerIdx = (currentPlayerIdx + 1) % util.NUM_PLAYERS
		time.Sleep(1 * time.Second)
		
	}
}

func (s *server) broadcast(sender *client, msg string) {
	for addr, member := range s.members {
		if sender != nil && addr == sender.conn.RemoteAddr() {
			continue
		}
		member.msg(msg)

	}
}

func (s *server) listCommands(sender *client) {
	msg := `available commands:
   /commands: list available commands
   /ready: be ready for the game
   /view: view your current cards
   /use <card1> <card2> ...: use the cards you selected
   /pass: pass your current turn
   /quit: quit the game`
	sender.msg(msg)
	sender.prompt()
}

func (s *server) quit(c *client) {
	c.msg("see you next time")
	s.broadcast(c, fmt.Sprintf("%s has left the room\n", c.nick))
	s.game.RemovePlayer(c.conn)
	log.Printf("client has disconnected: %s\n", c.nick)
	c.conn.Close()
}

func (s *server) ready(c *client) {
	player, _ := s.game.Players.Load(c.conn.RemoteAddr())
	player.(*util.Player).IsReady = true

	c.msg(fmt.Sprintf("you are ready for the game. %v/%v", s.game.NumReady(), util.NUM_PLAYERS))
	s.broadcast(c, fmt.Sprintf("%s is ready. %v/%v", c.nick, s.game.NumReady(), util.NUM_PLAYERS))
	if s.game.NumReady() == util.NUM_PLAYERS {
		s.broadcast(nil, "all players are ready. game will start soon...")
		time.Sleep(1 * time.Second)
	}
}

func (s *server) viewCards(c *client, args []string) {
	player, _ := s.game.Players.Load(c.conn.RemoteAddr())
	if player.(*util.Player).Position == util.LANDLORD {
		c.msg("your cards: " + player.(*util.Player).String() + " (landlord)")
	} else {
		c.msg("your cards: " + player.(*util.Player).String())
	}
}

func (s *server) useCards(c *client, args []string) {
	if s.game.CurrentPlayer.Conn.RemoteAddr() != c.conn.RemoteAddr() {
		c.err(errors.New("it's not your turn"))
		return
	}
	cardsString := args[1:]
	var cards []*util.Card
	var invalidCards []string
	for _, s := range cardsString {
		switch s {
		case "A":
			cards = append(cards, &util.Card{Point: util.ACE})
		case "2":
			cards = append(cards, &util.Card{Point: util.TWO})
		case "3":
			cards = append(cards, &util.Card{Point: util.THREE})
		case "4":
			cards = append(cards, &util.Card{Point: util.FOUR})
		case "5":
			cards = append(cards, &util.Card{Point: util.FIVE})
		case "6":
			cards = append(cards, &util.Card{Point: util.SIX})
		case "7":
			cards = append(cards, &util.Card{Point: util.SEVEN})
		case "8":
			cards = append(cards, &util.Card{Point: util.EIGHT})
		case "9":
			cards = append(cards, &util.Card{Point: util.NINE})
		case "10":
			cards = append(cards, &util.Card{Point: util.TEN})
		case "J":
			cards = append(cards, &util.Card{Point: util.JACK})
		case "Q":
			cards = append(cards, &util.Card{Point: util.QUEEN})
		case "K":
			cards = append(cards, &util.Card{Point: util.KING})
		case "joker":
			cards = append(cards, &util.Card{Point: util.BLACK_JOKER})
		case "JOKER":
			cards = append(cards, &util.Card{Point: util.RED_JOKER})
		default:
			invalidCards = append(invalidCards, s)
		}
	}
	if len(invalidCards) > 0 {
		c.err(errors.New(fmt.Sprintf("invalid cards: %v", invalidCards)))
		cmd := <-s.commands
		s.commands <- cmd
		return
	}
	if len(cards) == 0 {
		c.err(errors.New("please select at least one card"))
		cmd := <-s.commands
		s.commands <- cmd
		return
	}
	player, _ := s.game.Players.Load(c.conn.RemoteAddr())
	if player == s.game.LastPlayer {
		s.game.LastUsedCards = []*util.Card{}
	}
	lastCards := s.game.LastUsedCards
	err := player.(*util.Player).Use(cards, lastCards)
	log.Println(1)
	if err != nil {
		c.err(err)
	} else {
		s.game.CurrentUsedCards <- cards
		s.game.LastUsedCards = cards
		c.msg(fmt.Sprintf("you used the cards: %v", cards))
		s.broadcast(c, fmt.Sprintf("%s used the cards: %v", c.nick, cards))
	}

}

func (s *server) pass(c *client) {
	c.msg("you passed your turn")
	s.broadcast(c, fmt.Sprintf("%s passed their turn", c.nick))
	s.game.CurrentUsedCards <- []*util.Card{}
}

func lenSyncMap(m *sync.Map) int {
	count := 0
	m.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}
