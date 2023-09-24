package util

import (
	"errors"
	"log"
	"net"
	"strings"

	"golang.org/x/exp/slices"
)

type playerPosition int

const (
	LANDLORD playerPosition = iota
	FARMER
)

type Player struct {
	Cards    []*Card
	Conn     net.Conn
	Position playerPosition
	IsReady  bool
}

func NewPlayer(conn net.Conn) *Player {
	return &Player{
		[]*Card{},
		conn,
		FARMER,
		false,
	}
}

// Deal n cards from the deck.
func (p *Player) Deal(d *Deck, n int) {
	p.Cards = append(p.Cards, d.Deal(n)...)
	p.Sort()
}

func (p *Player) Use(cardsInfo []*Card, lastCardsInfo []*Card) error {
	if !Valid(cardsInfo) {
		return errors.New("invalid cards")
	}

	if !CompareTo(cardsInfo, lastCardsInfo) {
		return errors.New("cards can't beat last played cards")
	}

	if !Contains(p.Cards, cardsInfo) {
		return errors.New("you don't have the cards")
	}

	var removedCardsIdx []int

	for _, cardInfo := range cardsInfo {
		for i, card := range p.Cards {
			if !slices.Contains(removedCardsIdx, i) && cardInfo.Equal(*card) {
				removedCardsIdx = append(removedCardsIdx, i)
				cardInfo.Color = card.Color
				break
			}
		}
	}
	slices.Sort(removedCardsIdx)
	for i := len(removedCardsIdx) - 1; i >= 0; i-- {
		p.Cards = append(p.Cards[:removedCardsIdx[i]], p.Cards[removedCardsIdx[i]+1:]...)
	}

	p.Sort()
	log.Println(p.Cards)
	return nil
}

// Sort the player's card in a descending order.
func (p *Player) Sort() {
	slices.SortFunc(p.Cards, func(a, b *Card) int {
		switch {
		case a.Point < b.Point:
			return 1
		case a.Point == b.Point:
			return 0
		case a.Point > b.Point:
			return -1
		}
		return 0
	})
}

func (p *Player) String() string {
	var cards []string
	for i := 0; i < len(p.Cards); i++ {
		cards = append(cards, p.Cards[i].String())
	}
	return "[" + strings.Join(cards, ", ") + "]"
}
