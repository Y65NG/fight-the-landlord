package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"landlord/util"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type server struct {
	commands chan command
	// members  map[net.Addr]*client
	members sync.Map
	game    *util.Game
}

func newServer() *server {
	return &server{
		commands: make(chan command, 1),
		// members:  make(map[net.Addr]*client),
		members: sync.Map{},
		game:    util.NewGame(),
	}
}

func (s *server) newClient(conn net.Conn) {
	c := &client{
		commands: s.commands,
		conn:     conn,
	}
	// s.members[conn.RemoteAddr()] = c
	s.members.Store(conn.RemoteAddr(), c)
	c.msg("please set your nickname:")
	c.prompt()
	nickname, err := bufio.NewReader(conn).ReadString('\n')
	for nickname == "\n" || err != nil {
		c.err(errors.New("please type a valid name"))
		nickname, err = bufio.NewReader(conn).ReadString('\n')
	}
	nickname = strings.Trim(nickname, "\n")
	c.nick = nickname
	c.msg("______________________")
	time.Sleep(500 * time.Millisecond)
	c.msg("welcome to the server, " + c.nick + "\ntype /ready to join the game or /commands to see available commands")
	c.msg(fmt.Sprintf("online players: %v", lenSyncMap(&s.members)))
	c.prompt()
	s.broadcast(c, fmt.Sprintf("%s join the room", c.nick))
	c.readInput()
}

func (s *server) removeClosedClient() {
	for {
		s.members.Range(func(addr, c any) bool {
			addr, ok1 := addr.(net.Addr)
			client, ok2 := c.(*client)
			if ok1 && ok2 {
				_, err := client.conn.Write([]byte{})
				if err != nil && !(errors.Is(err, net.ErrClosed) &&
					errors.Is(err, io.EOF) &&
					errors.Is(err, syscall.EPIPE)) {
					log.Println("client has disconnected:", addr)
					// s.broadcast(nil, fmt.Sprintf("%s left the room", client.nick))
					client.conn.Close()
					s.members.Delete(addr)
				}
			}
			return true
		})
	}

}

func (s *server) runCommands() {
	for command := range s.commands {
		sender := command.sender
		sender.conn.SetDeadline(time.Now().Add(120 * time.Second))
		switch command.id {
		case CMD_EMPTY_LINE:
			if s.game.State != util.STATE_PLAYING || sender.conn.RemoteAddr() == s.game.CurrentPlayer.Conn.RemoteAddr() {
				sender.prompt()
			}
		case CMD_MESSAGE:
			s.broadcast(sender, sender.nick+": "+command.args[0])
			if s.game.State != util.STATE_PLAYING || sender.conn.RemoteAddr() == s.game.CurrentPlayer.Conn.RemoteAddr() {
				sender.prompt()
			}
		case CMD_LIST_COMMANDS:
			s.listCommands(sender)
		case CMD_LIST_PLAYERS:
			s.listPlayers(sender)
		case CMD_QUIT:
			s.quit(sender)
		case CMD_READY:
			if s.game.State != util.STATE_PLAYING {
				s.ready(sender)
			} else {
				sender.err(errors.New("you're already in a game"))
				if sender.conn.RemoteAddr() == s.game.CurrentPlayer.Conn.RemoteAddr() {
					sender.prompt()
				}
			}
		case CMD_VIEW_CARDS:
			if s.game.State == util.STATE_PLAYING && s.game.ContainsPlayer(sender.conn.RemoteAddr()) {
				s.viewCards(sender, command.args)
			} else {
				sender.err(errors.New("you must first join a game"))
				sender.prompt()
			}
		case CMD_USE_CARDS:
			if s.game.State == util.STATE_PLAYING && s.game.ContainsPlayer(sender.conn.RemoteAddr()) {
				s.useCards(sender, command.args)
			} else {
				sender.err(errors.New("you must first join a game"))
				sender.prompt()
			}
		case CMD_PASS:
			if s.game.State == util.STATE_PLAYING && s.game.ContainsPlayer(sender.conn.RemoteAddr()) {
				s.pass(sender)
			} else {
				sender.err(errors.New("you must first join a game"))
				sender.prompt()
			}
		case CMD_UNKNOWN:
			sender.err(errors.New("unknown command. Type /commands to see available commands"))
			if s.game.State == util.STATE_WAITING || sender.conn.RemoteAddr() == s.game.CurrentPlayer.Conn.RemoteAddr() {
				sender.prompt()
			}
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
		case util.STATE_OVER:
			s.broadcast(nil, "type /ready to start a new game or /quit to quit")
			s.game = util.NewGame()
			s.members.Range(func(_, c any) bool {
				s.game.AddPlayer(c.(*client).conn)
				return true
			})
		}
	}
}

func (s *server) play() {
	var players []*util.Player
	g := s.game
	g.Players.Range(func(_, player any) bool {
		if player, ok := player.(*util.Player); ok {
			player.Deal(&g.Deck, 17)
			player.Sort()
			players = append(players, player)
		}
		return true
	})

	landlordIdx := util.R.Intn(util.NUM_PLAYERS)
	g.Landlord = players[landlordIdx]
	g.Landlord.Deal(&g.Deck, 3)
	g.Landlord.Position = util.LANDLORD

	c, _ := s.members.Load(g.Landlord.Conn.RemoteAddr())
	c.(*client).msg("\b\byou are the landlord")
	s.broadcast(c.(*client), fmt.Sprintf("%s is the landlord", c.(*client).nick))

	currentPlayerIdx := landlordIdx
	time.Sleep(1 * time.Second)
	s.members.Range(func(_, c any) bool {
		if player, ok := g.Players.Load(c.(*client).conn.RemoteAddr()); ok {
			if player.(*util.Player).Position != util.LANDLORD {
				c.(*client).msg("\b\byour cards: \n" + player.(*util.Player).String() + " (" + strconv.Itoa(len(player.(*util.Player).Cards)) + " remaining)")
				// s.members[c.conn.RemoteAddr()].msg("\b\byour cards: \n" + player.(*util.Player).String() + " (" + strconv.Itoa(len(player.(*util.Player).Cards)) + " remaining)")
			}
		}
		return true
	})
	for {
		g.CurrentPlayer = players[currentPlayerIdx]
		if g.CurrentPlayer == g.LastPlayer {
			g.LastUsedCards = []*util.Card{}
		}
		c, _ := s.members.Load(g.CurrentPlayer.Conn.RemoteAddr())
		c.(*client).msg("it's your turn")
		s.viewCards(c.(*client), []string{})
		s.broadcast(c.(*client), fmt.Sprintf("waiting for %s's action...", c.(*client).nick))

		cards := <-g.CurrentUsedCards
		if g.PlayerNum != g.NumPlayers {
			break
		}
		if len(g.CurrentPlayer.Cards) == 0 {
			s.broadcast(nil, fmt.Sprintf("%s won the game", c.(*client).nick))
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
	log.Println("game suddenly ends")
}

func (s *server) broadcast(sender *client, msg string) {
	s.members.Range(func(addr, member any) bool {
		if sender != nil && addr == sender.conn.RemoteAddr() {
			return true
		}
		member.(*client).msg("\b\b" + msg)
		if s.game.State != util.STATE_PLAYING || (s.game.CurrentPlayer != nil && addr == s.game.CurrentPlayer.Conn.RemoteAddr()) {
			member.(*client).prompt()
		}
		return true
	})
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
	if s.game.State == util.STATE_WAITING || sender.conn.RemoteAddr() == s.game.CurrentPlayer.Conn.RemoteAddr() {
		sender.prompt()
	}
}

func (s *server) listPlayers(sender *client) {
	sender.msg("online players:")
	s.members.Range(func(_, c any) bool {
		sender.msg(c.(*client).nick)
		return true
	})
	if s.game.State == util.STATE_WAITING || sender.conn.RemoteAddr() == s.game.CurrentPlayer.Conn.RemoteAddr() {
		sender.prompt()
	}
}

func (s *server) quit(c *client) {
	defer c.conn.Close()
	c.msg("see you next time")
	if ok := s.game.RemovePlayer(c.conn); ok {
		if s.game.State == util.STATE_PLAYING {
			s.game.NextState()
			s.broadcast(c, fmt.Sprintf("%s left the room, game ends", c.nick))
			s.game.CurrentUsedCards <- []*util.Card{}
		} else {
			s.broadcast(c, fmt.Sprintf("%s left the room", c.nick))
		}
	} else {
		s.broadcast(c, fmt.Sprintf("%s left the room", c.nick))
	}
	s.members.Delete(c.conn.RemoteAddr())
	log.Printf("client has disconnected: %s\n", c.nick)
}

func (s *server) ready(c *client) {
	s.game.AddPlayer(c.conn)
	player, _ := s.game.Players.Load(c.conn.RemoteAddr())
	player.(*util.Player).IsReady = true

	c.msg(fmt.Sprintf("you are ready for the game. %v/%v", s.game.NumReady(), s.game.NumPlayers))
	c.prompt()
	s.broadcast(c, fmt.Sprintf("%s is ready. %v/%v", c.nick, s.game.NumReady(), s.game.NumPlayers))
	if s.game.NumReady() == util.NUM_PLAYERS {
		s.broadcast(nil, "all players are ready. game will start soon...")
		time.Sleep(1 * time.Second)
	}
}

func (s *server) viewCards(c *client, args []string) {
	player, _ := s.game.Players.Load(c.conn.RemoteAddr())
	c.msg(fmt.Sprintf("cards you have to beat: %v", s.game.LastUsedCards))
	if player.(*util.Player).Position == util.LANDLORD {
		c.msg("your cards:\n" + player.(*util.Player).String() + " (landlord)")
	} else {
		c.msg("your cards:\n" + player.(*util.Player).String())
	}
	if s.game.CurrentPlayer.Conn.RemoteAddr() == c.conn.RemoteAddr() {
		c.prompt()
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
		switch strings.ToUpper(s) {
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
		case "JOKER":
			if s == "joker" {
				cards = append(cards, &util.Card{Point: util.BLACK_JOKER})
			} else {
				cards = append(cards, &util.Card{Point: util.RED_JOKER})
			}
		default:
			invalidCards = append(invalidCards, s)
		}
	}
	if len(invalidCards) > 0 {
		c.err(errors.New(fmt.Sprintf("invalid cards: %v", invalidCards)))
		c.prompt()
		cmd := <-s.commands
		s.commands <- cmd
		return
	}
	if len(cards) == 0 {
		c.err(errors.New("please select at least one card"))
		c.prompt()
		cmd := <-s.commands
		s.commands <- cmd
		return
	}
	player, _ := s.game.Players.Load(c.conn.RemoteAddr())
	lastCards := s.game.LastUsedCards
	err := player.(*util.Player).Use(cards, lastCards)
	if err != nil {
		c.err(err)
		c.prompt()
	} else {
		s.game.CurrentUsedCards <- cards
		s.game.LastUsedCards = cards
		s.game.LastPlayer = player.(*util.Player)
		c.msg(fmt.Sprintf("you used the cards: %v", cards))
		s.broadcast(c, fmt.Sprintf("%s used the cards: %v (%v remaining)", c.nick, cards, len(s.game.CurrentPlayer.Cards)))
	}

}

func (s *server) pass(c *client) {
	if s.game.CurrentPlayer.Conn.RemoteAddr() != c.conn.RemoteAddr() {
		c.err(errors.New("it's not your turn"))
		return
	}
	c.msg("you passed your turn")
	s.broadcast(c, fmt.Sprintf("%s passed their turn", c.nick))
	s.game.CurrentUsedCards <- []*util.Card{}
}


func (s *server) SetNumPlayers(n int) {
	s.game.NumPlayers = n
}

func lenSyncMap(m *sync.Map) int {
	count := 0
	m.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}
