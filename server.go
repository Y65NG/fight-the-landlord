package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"landlord/util"
	"log"
	"net"
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
		Nick:     "anonymous",
		commands: s.commands,
		Conn:     conn,
	}
	// s.members[conn.RemoteAddr()] = c
	s.members.Store(conn.RemoteAddr(), c)
	nickname, err := bufio.NewReader(conn).ReadString('\n')
	for nickname == "\n" || err != nil {
		c.Conn.Write([]byte(""))
		nickname, err = bufio.NewReader(conn).ReadString('\n')
	}
	c.Conn.Write([]byte("ok\n"))
	nickname = strings.Trim(nickname, "\n")
	c.Nick = nickname
	time.Sleep(100 * time.Millisecond)
	c.msg(MSG_MESSAGE, "welcome to the server, "+c.Nick+"\ntype /ready to join the games")
	// c.msg(MSG_MESSAGE, fmt.Sprintf("online players: %v", lenSyncMap(&s.members)))
	s.broadcast(MSG_MESSAGE, c, fmt.Sprintf("%s join the room", c.Nick))
	s.broadcast(MSG_ROOM_INFO, nil, util.State(s.game.State)+"_"+strings.Join(s.listPlayers(), "\n"))
	c.readInput()
}

func (s *server) removeClosedClient() {
	for {
		s.members.Range(func(addr, c any) bool {
			addr, ok1 := addr.(net.Addr)
			client, ok2 := c.(*client)
			if ok1 && ok2 {
				_, err := client.Conn.Write([]byte{})
				if err != nil && !(errors.Is(err, net.ErrClosed) &&
					errors.Is(err, io.EOF) &&
					errors.Is(err, syscall.EPIPE)) {
					log.Println("client has disconnected:", addr)
					// s.broadcast(nil, fmt.Sprintf("%s left the room", client.nick))
					client.Conn.Close()
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
		sender.Conn.SetDeadline(time.Now().Add(300 * time.Second))
		switch command.id {
		case CMD_EMPTY_LINE:
			if s.game.State != util.STATE_PLAYING || sender.Conn.RemoteAddr() == s.game.CurrentPlayer.Conn.RemoteAddr() {
				// sender.prompt()
			}
		case CMD_MESSAGE:
			sender.msg(MSG_CHAT, sender.Nick+": "+command.args[0])
			s.broadcast(MSG_CHAT, sender, sender.Nick+": "+command.args[0])
			if s.game.State != util.STATE_PLAYING || sender.Conn.RemoteAddr() == s.game.CurrentPlayer.Conn.RemoteAddr() {
				// sender.prompt()
			}
		case CMD_LIST_COMMANDS:
			s.listCommands(sender)
		case CMD_LIST_PLAYERS:
			s.listPlayers()
		case CMD_QUIT:
			s.quit(sender)
		case CMD_READY:
			if s.game.State != util.STATE_PLAYING {
				s.ready(sender)
			} else {
				sender.err(errors.New("you're already in a game"))
			}
		case CMD_VIEW_CARDS:
			if s.game.State == util.STATE_PLAYING && s.game.ContainsPlayer(sender.Conn.RemoteAddr()) {
				s.viewCards(sender, command.args)
			} else {
				sender.err(errors.New("you must first join a game"))
				// sender.prompt()
			}
		case CMD_USE_CARDS:
			if s.game.State == util.STATE_PLAYING && s.game.ContainsPlayer(sender.Conn.RemoteAddr()) {
				s.useCards(sender, command.args)
			} else {
				sender.err(errors.New("you must first join a game"))
				// sender.prompt()
			}
		case CMD_PASS:
			if s.game.State == util.STATE_PLAYING && s.game.ContainsPlayer(sender.Conn.RemoteAddr()) {
				s.pass(sender)
			} else {
				sender.err(errors.New("you must first join a game"))
				// sender.prompt()
			}
		case CMD_UNKNOWN:
			sender.err(errors.New("unknown command. Type /commands to see available commands"))
			if s.game.State == util.STATE_WAITING || sender.Conn.RemoteAddr() == s.game.CurrentPlayer.Conn.RemoteAddr() {
				// sender.prompt()
			}
		}
	}
}

func (s *server) gameLoop() {
	for {
		switch s.game.State {
		case util.STATE_WAITING:
			if s.game.NumReady() == s.game.NumPlayers {
				s.game.NextState()
			}
		case util.STATE_PLAYING:
			time.Sleep(1 * time.Second)
			s.play()
		case util.STATE_OVER:
			s.broadcast(MSG_MESSAGE, nil, "type /ready to start a new game or /quit to quit")
			numPlayers := s.game.NumPlayers
			s.game = util.NewGame()
			s.game.NumPlayers = numPlayers
			s.members.Range(func(_, c any) bool {
				s.game.AddPlayer(c.(*client).Conn, c.(*client).Nick)
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

	time.Sleep(1 * time.Second)

	landlordIdx := util.R.Intn(g.NumPlayers)
	g.Landlord = players[landlordIdx]
	g.Landlord.Deal(&g.Deck, 3)
	g.Landlord.Position = util.LANDLORD

	c, ok := s.members.Load(g.Landlord.Conn.RemoteAddr())
	if !ok {
		log.Println("unable to load client")
		g.NextState()
		return
	}
	c.(*client).msg(MSG_MESSAGE, "\b\byou are the landlord")
	s.broadcast(MSG_MESSAGE, c.(*client), fmt.Sprintf("%s is the landlord", c.(*client).Nick))
	s.broadcast(MSG_ROOM_INFO, nil, util.State(s.game.State)+"_"+strings.Join(s.listPlayers(), "\n"))
	currentPlayerIdx := landlordIdx
	time.Sleep(1 * time.Second)
	s.members.Range(func(_, c any) bool {
		if player, ok := g.Players.Load(c.(*client).Conn.RemoteAddr()); ok {
			if player.(*util.Player).Position != util.LANDLORD {
				c.(*client).msg(MSG_PLAYER_STATUS, "farmer_"+player.(*util.Player).String())
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
		if _, ok := c.(*client); !ok {
			log.Println("unable to load client")
			g.NextState()
			break
		}
		s.broadcast(MSG_ROOM_INFO, nil, util.State(s.game.State)+"_"+strings.Join(s.listPlayers(), "\n"))
		statusStr := g.CurrentPlayer.Highlight(g.LastUsedCards)
		if g.CurrentPlayer.Position == util.LANDLORD {
			statusStr = "landlord_" + statusStr
		} else {
			statusStr = "farmer_" + statusStr
		}
		c.(*client).msg(MSG_INFO, "it's your turn")
		if len(g.LastUsedCards) > 0 {
			c.(*client).msg(MSG_INFO, fmt.Sprintf("you have to beat %v from %v", g.LastUsedCards, g.LastPlayer.Nick))
		} else {
			c.(*client).msg(MSG_INFO, "you can play any cards")
		}
		s.viewCards(c.(*client), []string{})
		s.broadcast(MSG_INFO, c.(*client), fmt.Sprintf("waiting for %s's action...", c.(*client).Nick))

		cards := <-g.CurrentUsedCards
		if g.PlayerNum != g.NumPlayers {
			break
		}
		if len(g.CurrentPlayer.Cards) == 0 {
			s.broadcast(MSG_MESSAGE, nil, fmt.Sprintf("%s won the game", c.(*client).Nick))
			g.NextState()
			break
		}
		if len(cards) == 0 {
			currentPlayerIdx = (currentPlayerIdx + 1) % g.NumPlayers
			continue
		}

		currentPlayerIdx = (currentPlayerIdx + 1) % g.NumPlayers
		// time.Sleep(1 * time.Second)

	}
	log.Println("game suddenly ends")
}

func (s *server) broadcast(msgType messageType, sender *client, msg string) {
	s.members.Range(func(addr, member any) bool {
		if sender != nil && addr == sender.Conn.RemoteAddr() {
			return true
		}
		if member.(*client).Nick == "anonymous" {
			return true
		}
		member.(*client).msg(msgType, msg)
		if s.game.State != util.STATE_PLAYING || (s.game.CurrentPlayer != nil && addr == s.game.CurrentPlayer.Conn.RemoteAddr()) {
			// member.(*client).prompt()
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
	sender.msg(MSG_MESSAGE, msg)
	if s.game.State == util.STATE_WAITING || sender.Conn.RemoteAddr() == s.game.CurrentPlayer.Conn.RemoteAddr() {
		// sender.prompt()
	}
}

func (s *server) listPlayers() []string {
	var players []string
	s.members.Range(func(_, c any) bool {
		switch s.game.State {
		case util.STATE_WAITING:
			if _, ok := s.game.Players.Load(c.(*client).Conn.RemoteAddr()); ok {
				players = append(players, " - "+c.(*client).Nick+" (ready)")
			} else {
				players = append(players, " - "+c.(*client).Nick)
			}
		case util.STATE_PLAYING:
			player, ok := s.game.Players.Load(c.(*client).Conn.RemoteAddr())
			if ok {
				playerStr := " -"
				if s.game.CurrentPlayer != nil && player.(*util.Player).Conn.RemoteAddr() == s.game.CurrentPlayer.Conn.RemoteAddr() {
					playerStr += ">"
				}
				playerStr += " " + c.(*client).Nick
				if player.(*util.Player).Position == util.LANDLORD {
					playerStr += " (landlord)"
				}
				players = append(players, playerStr)
			}
		case util.STATE_OVER:
			players = append(players, c.(*client).Nick)
		}
		return true
	})

	return players
}

func (s *server) quit(c *client) {
	defer c.Conn.Close()
	c.msg(MSG_MESSAGE, "see you next time")
	if ok := s.game.RemovePlayer(c.Conn); ok {
		if s.game.State == util.STATE_PLAYING {
			s.game.NextState()
			s.broadcast(MSG_MESSAGE, c, fmt.Sprintf("%s left the room, game ends", c.Nick))
			s.game.CurrentUsedCards <- []*util.Card{}
		} else {
			s.broadcast(MSG_MESSAGE, c, fmt.Sprintf("%s left the room", c.Nick))
		}
	} else {
		s.broadcast(MSG_MESSAGE, c, fmt.Sprintf("%s left the room", c.Nick))
	}
	s.members.Delete(c.Conn.RemoteAddr())
	log.Printf("client has disconnected: %s (%v)\n", c.Nick, c.Conn.RemoteAddr())
}

func (s *server) ready(c *client) {
	s.game.AddPlayer(c.Conn, c.Nick)
	player, _ := s.game.Players.Load(c.Conn.RemoteAddr())
	if player.(*util.Player).IsReady {
		c.err(errors.New("you're already ready"))
		return
	}
	player.(*util.Player).IsReady = true

	c.msg(MSG_MESSAGE, fmt.Sprintf("you are ready for the game. %v/%v", s.game.NumReady(), s.game.NumPlayers))

	// c.prompt()
	s.broadcast(MSG_MESSAGE, c, fmt.Sprintf("%s is ready. %v/%v", c.Nick, s.game.NumReady(), s.game.NumPlayers))
	s.broadcast(MSG_ROOM_INFO, nil, util.State(s.game.State)+"_"+strings.Join(s.listPlayers(), "\n"))
	if s.game.NumReady() == s.game.NumPlayers {
		s.broadcast(MSG_MESSAGE, nil, "all players are ready. game will start soon...")
		time.Sleep(1 * time.Second)
	}
}

func (s *server) viewCards(c *client, args []string) {
	player, _ := s.game.Players.Load(c.Conn.RemoteAddr())
	msg := player.(*util.Player).Highlight(s.game.LastUsedCards)
	if player.(*util.Player).Position == util.LANDLORD {
		msg = "landlord_" + msg
	} else {
		msg = "farmer_" + msg
	}
	if len(player.(*util.Player).Recommend(s.game.LastUsedCards)) == 0 {
		c.msg(MSG_INFO, "you have no cards to beat the last player")
	}
	c.msg(MSG_PLAYER_STATUS, msg)
}

func (s *server) useCards(c *client, args []string) {
	if s.game.CurrentPlayer.Conn.RemoteAddr() != c.Conn.RemoteAddr() {
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
		// c.prompt()
		cmd := <-s.commands
		s.commands <- cmd
		return
	}
	if len(cards) == 0 {
		c.err(errors.New("please select at least one card"))
		// c.prompt()
		cmd := <-s.commands
		s.commands <- cmd
		return
	}
	player, _ := s.game.Players.Load(c.Conn.RemoteAddr())
	lastCards := s.game.LastUsedCards
	err := player.(*util.Player).Use(cards, lastCards)
	if err != nil {
		c.err(err)
		// c.prompt()
	} else {
		s.game.CurrentUsedCards <- cards
		s.game.LastUsedCards = cards
		s.game.LastPlayer = player.(*util.Player)
		c.msg(MSG_MESSAGE, fmt.Sprintf("you used the cards: %v", cards))
		s.broadcast(MSG_MESSAGE, c, fmt.Sprintf("%s used the cards: %v (%v remaining)", c.Nick, cards, len(s.game.CurrentPlayer.Cards)))
	}

}

func (s *server) pass(c *client) {
	if s.game.CurrentPlayer.Conn.RemoteAddr() != c.Conn.RemoteAddr() {
		c.err(errors.New("it's not your turn"))
		return
	}
	c.msg(MSG_MESSAGE, "you passed your turn")
	s.broadcast(MSG_MESSAGE, c, fmt.Sprintf("%s passed their turn", c.Nick))
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
