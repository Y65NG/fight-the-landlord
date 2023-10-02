package util

import (
	"errors"
	"fmt"
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
	Nick     string
	Position playerPosition
	IsReady  bool
}

func NewPlayer(conn net.Conn, nick string) *Player {
	return &Player{
		[]*Card{},
		conn,
		nick,
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
	Sort(cardsInfo)
	Sort(lastCardsInfo)
	if !Valid(cardsInfo) {
		return errors.New("invalid cards")
	}

	if !Contains(p.Cards, cardsInfo) {
		return errors.New("you don't have the cards")
	}

	if !CompareTo(cardsInfo, lastCardsInfo) {
		return errors.New("cards can't beat last played cards")
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
	return nil
}

func (p *Player) Recommend(lastCards []*Card) []*Card {
	lenth := len(lastCards)
	if lenth == 0 {
		lenth++
	}
	idx := len(p.Cards) - 1
	for idx >= lenth-1 {
		cards := p.Cards[idx-lenth+1 : idx+1]
		if Valid(cards) && CompareTo(cards, lastCards) {
			if !isBomb(cards) {
				return cards
			} else {
				idx -= 3
			}
		}
		idx--
	}
	for i := len(p.Cards) - 1; i >= lenth-1; i-- {
		cards := p.Cards[i-lenth+1 : i+1]
		if isBomb(cards) {
			return cards
		}
	}
	return []*Card{}
}

func (p *Player) Score(low, high int) int {
	score := 0
	cards := p.Cards[low:high]
	switch {
	case isBomb(cards):
		score += 1000
	case isPlane(cards):
		score += 900
	case isDoubleStraight(cards):
		score += 800
	case isStraight(cards):
		score += 700
	case isTripleWithTwo(cards):
		score += 600
	case isTripleWithOne(cards):
		score += 500
	case isTriple(cards):
		score += 400
	case isDouble(cards):
		score += 300
	case isSingle(cards):
		score += 200
	}
	score += int(cards[0].Point)
	return score
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

func (p *Player) Highlight(lastCards []*Card) string {
	var cards []string
	recommends := p.Recommend(lastCards)
	numSelection := 1
	for i := 0; i < len(p.Cards); i++ {
		if slices.ContainsFunc(recommends, func(c *Card) bool {
			return c.Point == p.Cards[i].Point && c.Color == p.Cards[i].Color
		}) {
			// cards = append(cards, color.InBold(color.InCyan(p.Cards[i].String())))
			cards = append(cards, fmt.Sprintf(`["%v"]%s[""]`,  numSelection, p.Cards[i].String()))
			numSelection++
		} else {
			cards = append(cards, p.Cards[i].String())
		}
	}

	return "[" + strings.Join(cards, ", ") + "]"
}
