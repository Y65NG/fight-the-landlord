package server

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"landlord/server/util"
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

func NewServer() *server {
	return &server{
		commands: make(chan command, 1),
		// members:  make(map[net.Addr]*client),
		members: sync.Map{},
		game:    util.NewGame(),
	}
}

func (s *server) NewClient(conn net.Conn) {
	c := &client{
		Nick:     "#anonymous",
		commands: s.commands,
		Conn:     conn,
	}
	// s.members[conn.RemoteAddr()] = c
	s.members.Store(conn.RemoteAddr(), c)
	nickname, err := bufio.NewReader(conn).ReadString('\n')
	nickname = strings.Trim(nickname, "\r\n")
	for nickname == "" || err != nil {
		c.Conn.Write([]byte(""))
		nickname, err = bufio.NewReader(conn).ReadString('\n')
		nickname = strings.Trim(nickname, "\r\n")
	}
	c.Conn.Write([]byte("ok\n"))
	nickname = strings.Trim(nickname, "\n")
	c.Nick = nickname
	time.Sleep(500 * time.Millisecond)
	c.msg(MSG_MESSAGE, "> welcome to the server, "+c.Nick+"\n  type /ready to join the games")
	s.broadcast(MSG_MESSAGE, c, fmt.Sprintf("> %s join the room", c.Nick))
	s.broadcast(MSG_ROOM_INFO, nil, util.State(s.game.State)+"_"+strings.Join(s.listPlayers(), "\n"))
	c.readInput()
}

func (s *server) RemoveClosedClient() {
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
					client.Conn.Close()
					s.members.Delete(addr)
				}
			}
			return true
		})
	}

}

func (s *server) RunCommands() (err error) {
	for command := range s.commands {
		sender := command.sender
		sender.Conn.SetDeadline(time.Now().Add(300 * time.Second))
		switch command.id {
		case CMD_MESSAGE:
			err = sender.msg(MSG_CHAT, sender.Nick+": "+command.args[0])
			if err != nil {
				return err
			}
			s.broadcast(MSG_CHAT, sender, sender.Nick+": "+command.args[0])
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
				sender.err(errors.New("> you're already in a game"))
			}
		case CMD_VIEW_CARDS:
			if s.game.State == util.STATE_PLAYING && s.game.ContainsPlayer(sender.Conn.RemoteAddr()) {
				s.viewCards(sender, command.args)
			} else {
				sender.err(errors.New("> you must first join a game"))
			}
		case CMD_USE_CARDS:
			if s.game.State == util.STATE_PLAYING && s.game.ContainsPlayer(sender.Conn.RemoteAddr()) {
				s.useCards(sender, command.args)
			} else {
				sender.err(errors.New("> you must first join a game"))
			}
		case CMD_PASS:
			if s.game.State == util.STATE_PLAYING && s.game.ContainsPlayer(sender.Conn.RemoteAddr()) {
				s.pass(sender)
			} else {
				sender.err(errors.New("> you must first join a game"))
			}
		case CMD_UNKNOWN:
			sender.err(errors.New("> unknown command: " + command.args[0]))

		}
	}
	return
}

func (s *server) GameLoop() (err error) {
	for {
		switch s.game.State {
		case util.STATE_WAITING:
			if s.game.NumReady() == s.game.NumPlayers {
				s.game.NextState()
			}
		case util.STATE_PLAYING:
			time.Sleep(1 * time.Second)
			err = s.play()
			if err != nil {
				log.Println(err)
				return
			}
		case util.STATE_OVER:
			time.Sleep(500 * time.Millisecond)
			s.broadcast(MSG_MESSAGE, nil, "> type /ready to start a new game or /quit to quit")
			numPlayers := s.game.NumPlayers
			s.game = util.NewGame()
			s.game.NumPlayers = numPlayers
			s.broadcast(MSG_ROOM_INFO, nil, util.State(s.game.State)+"_"+strings.Join(s.listPlayers(), "\n"))
		}
	}
}

func (s *server) play() (err error) {
	var players []*util.Player
	g := s.game
	g.Players.Range(func(_, player any) bool {
		if player, ok := player.(*util.Player); ok {
			err = player.Deal(&g.Deck, 17)
			if err != nil {
				return false
			}
			player.Sort()
			players = append(players, player)
		}
		return true
	})
	if err != nil {
		return err
	}

	time.Sleep(500 * time.Millisecond)

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
	c.(*client).msg(MSG_MESSAGE, "> you are the landlord")
	s.broadcast(MSG_MESSAGE, c.(*client), fmt.Sprintf("> %s is the landlord", c.(*client).Nick))
	s.broadcast(MSG_ROOM_INFO, nil, util.State(s.game.State)+"_"+strings.Join(s.listPlayers(), "\n"))
	currentPlayerIdx := landlordIdx
	time.Sleep(500 * time.Millisecond)
	s.members.Range(func(_, c any) bool {
		if player, ok := g.Players.Load(c.(*client).Conn.RemoteAddr()); ok {
			if player.(*util.Player).Position != util.LANDLORD {
				err = c.(*client).msg(MSG_PLAYER_STATUS, "farmer_"+player.(*util.Player).String())
				if err != nil {
					return false
				}
			}
		}
		return true
	})
	if err != nil {
		return err
	}

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
		c.(*client).msg(MSG_INFO, "> it's your turn")
		time.Sleep(300 * time.Millisecond)
		if len(g.LastUsedCards) > 0 {
			c.(*client).msg(MSG_INFO, fmt.Sprintf("  you have to beat %v from %v", util.CardsToString(g.LastUsedCards), g.LastPlayer.Nick))
		} else {
			c.(*client).msg(MSG_INFO, "  you can play any cards")
		}
		s.viewCards(c.(*client), []string{})
		if len(g.CurrentPlayer.Recommend(s.game.LastUsedCards)) == 0 {
			log.Println(s.game.LastUsedCards)
			c.(*client).msg(MSG_INFO, "> you can't beat the last player")
		} else {
			c.(*client).msg(MSG_INFO, fmt.Sprintf("> recommend: %v", g.CurrentPlayer.Recommend(g.LastUsedCards)))
		}
		s.broadcast(MSG_INFO, c.(*client), fmt.Sprintf("> waiting for %s's action...", c.(*client).Nick))

		cards := <-g.CurrentUsedCards
		if g.PlayerNum != g.NumPlayers {
			log.Println("ln 237")
			log.Println(g.PlayerNum, g.NumPlayers)
			break
		}
		if len(g.CurrentPlayer.Cards) == 0 {
			s.broadcast(MSG_MESSAGE, nil, fmt.Sprintf("> %s won the game", c.(*client).Nick))
			g.NextState()
			break
		}
		if len(cards) == 0 {
			currentPlayerIdx = (currentPlayerIdx + 1) % g.NumPlayers
			continue
		}

		currentPlayerIdx = (currentPlayerIdx + 1) % g.NumPlayers
		time.Sleep(500 * time.Millisecond)

	}
	log.Println("game ends")
	return
}

func (s *server) broadcast(msgType messageType, sender *client, msg string) {
	s.members.Range(func(addr, member any) bool {
		if sender != nil && addr == sender.Conn.RemoteAddr() {
			return true
		}
		if member.(*client).Nick == "#anonymous" {
			return true
		}
		member.(*client).msg(msgType, msg)
		if s.game.State != util.STATE_PLAYING || (s.game.CurrentPlayer != nil && addr == s.game.CurrentPlayer.Conn.RemoteAddr()) {
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
}

func (s *server) listPlayers() []string {
	var players []string
	s.members.Range(func(_, c any) bool {
		if c.(*client).Nick == "#anonymous" {
			return true
		}
		switch s.game.State {
		case util.STATE_WAITING:
			if player, ok := s.game.Players.Load(c.(*client).Conn.RemoteAddr()); ok && player.(*util.Player).IsReady {
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
	c.msg(MSG_STOP, "> see you next time")
	if ok := s.game.RemovePlayer(c.Conn); ok {
		if s.game.State == util.STATE_PLAYING {
			s.game.NextState()
			s.broadcast(MSG_MESSAGE, c, fmt.Sprintf("> %s left the room, game ends", c.Nick))
			s.game.CurrentUsedCards <- []*util.Card{}
		} else {
			s.broadcast(MSG_MESSAGE, c, fmt.Sprintf("> %s left the room", c.Nick))
		}
	} else {
		s.broadcast(MSG_MESSAGE, c, fmt.Sprintf("> %s left the room", c.Nick))
	}
	s.members.Delete(c.Conn.RemoteAddr())
	log.Printf("client has disconnected: %s (%v)\n", c.Nick, c.Conn.RemoteAddr())
}

func (s *server) ready(c *client) {
	s.game.AddPlayer(c.Conn, c.Nick)
	player, _ := s.game.Players.Load(c.Conn.RemoteAddr())
	if player.(*util.Player).IsReady {
		c.err(errors.New("> you're already ready"))
		return
	}
	player.(*util.Player).IsReady = true

	c.msg(MSG_MESSAGE, fmt.Sprintf("> you are ready for the game. %v/%v", s.game.NumReady(), s.game.NumPlayers))

	// c.prompt()
	s.broadcast(MSG_MESSAGE, c, fmt.Sprintf("> %s is ready. %v/%v", c.Nick, s.game.NumReady(), s.game.NumPlayers))
	s.broadcast(MSG_ROOM_INFO, nil, util.State(s.game.State)+"_"+strings.Join(s.listPlayers(), "\n"))
	if s.game.NumReady() == s.game.NumPlayers {
		s.broadcast(MSG_MESSAGE, nil, "> all players are ready. game will start soon...")
		time.Sleep(1 * time.Second)
	}
}

func (s *server) viewCards(c *client, args []string) {
	player, _ := s.game.Players.Load(c.Conn.RemoteAddr())
	if _, ok := player.(*util.Player); !ok {
		return
	}
	msg := player.(*util.Player).Highlight(s.game.LastUsedCards)
	if player.(*util.Player).Position == util.LANDLORD {
		msg = "landlord_" + msg
	} else {
		msg = "farmer_" + msg
	}

	c.msg(MSG_PLAYER_STATUS, msg)
}

func (s *server) useCards(c *client, args []string) {
	if s.game.CurrentPlayer.Conn.RemoteAddr() != c.Conn.RemoteAddr() {
		c.err(errors.New("> it's not your turn"))
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
		c.err(errors.New(fmt.Sprintf("> invalid cards: %v", invalidCards)))
		cmd := <-s.commands
		s.commands <- cmd
		return
	}
	if len(cards) == 0 {
		c.err(errors.New("> please select at least one card"))
		cmd := <-s.commands
		s.commands <- cmd
		return
	}
	player, _ := s.game.Players.Load(c.Conn.RemoteAddr())
	lastCards := s.game.LastUsedCards
	err := player.(*util.Player).Use(cards, lastCards)
	if err != nil {
		c.err(err)
	} else {
		s.game.CurrentUsedCards <- cards
		s.game.LastUsedCards = cards
		s.game.LastPlayer = player.(*util.Player)
		c.msg(MSG_MESSAGE, fmt.Sprintf("> you used the cards: %v", cards))
		s.viewCards(c, []string{})
		s.broadcast(MSG_MESSAGE, c, fmt.Sprintf("> %s used the cards: %v (%v remaining)", c.Nick, cards, len(s.game.CurrentPlayer.Cards)))
	}

}

func (s *server) pass(c *client) {
	if s.game.CurrentPlayer.Conn.RemoteAddr() != c.Conn.RemoteAddr() {
		c.err(errors.New("> it's not your turn"))
		return
	}
	c.msg(MSG_MESSAGE, "> you passed your turn")
	s.broadcast(MSG_MESSAGE, c, fmt.Sprintf("> %s passed their turn", c.Nick))
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
