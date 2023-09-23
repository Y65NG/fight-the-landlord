package util

import (
	"net"
	"sync"
)

// import "fmt"
type GameState int

const (
	STATE_WAITING GameState = iota
	STATE_PLAYING
	STATE_OVER
)

const NUM_PLAYERS = 2

type Game struct {
	// Players          map[net.Addr]*Player
	Players          sync.Map
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
		Deck:             deck,
		State:            STATE_WAITING,
		CurrentUsedCards: make(chan []*Card, 1),
	}
}

func (g *Game) AddPlayer(conn net.Conn) {
	g.Players.Store(conn.RemoteAddr(), NewPlayer(conn))
}

func (g *Game) RemovePlayer(conn net.Conn) {
	g.Players.Delete(conn.RemoteAddr())
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
