package util

import (
	"net"
	"sync"
)

const NUM_PLAYERS = 3

type GameState int

const (
	STATE_WAITING GameState = iota
	STATE_PLAYING
	STATE_OVER
)

type Game struct {
	// Players          map[net.Addr]*Player
	Players          sync.Map
	NumPlayers       int
	PlayerNum        int
	Deck             Deck
	State            GameState
	CurrentUsedCards chan []*Card
	LastUsedCards    []*Card
	LastPlayer       *Player
	CurrentPlayer    *Player
	Landlord         *Player
}

func NewGame() *Game {
	deck := NewDeck()
	deck.Shuffle()
	return &Game{
		Players:          sync.Map{},
		NumPlayers:       NUM_PLAYERS,
		PlayerNum:        0,
		Deck:             deck,
		State:            STATE_WAITING,
		CurrentUsedCards: make(chan []*Card, 1),
	}
}

func (g *Game) AddPlayer(conn net.Conn) {
	g.Players.Store(conn.RemoteAddr(), NewPlayer(conn))
	g.PlayerNum++
}

func (g *Game) RemovePlayer(conn net.Conn) bool {
	if _, ok := g.Players.LoadAndDelete(conn.RemoteAddr()); ok {
		g.PlayerNum--
		return true
	}
	return false
}

func (g *Game) ContainsPlayer(addr net.Addr) bool {
	_, ok := g.Players.Load(addr)
	return ok
}


func (g *Game) NextState() {
	switch g.State {
	case STATE_WAITING:
		g.State = STATE_PLAYING
	case STATE_PLAYING:
		g.State = STATE_OVER
	case STATE_OVER:
		g.State = STATE_WAITING
	}
}

func (g *Game) NumReady() int {
	count := 0
	// for _, player := range players {
	g.Players.Range(func(_, player any) bool {
		if player, ok := player.(*Player); ok {
			if player.IsReady {
				count++
			}
		}
		return true
	})
	return count
}
