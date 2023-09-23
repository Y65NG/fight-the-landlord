package util

import (
	"fmt"
	"math/rand"
	"time"
)

const NUM_CARDS = 54

var R = rand.New(rand.NewSource(time.Now().UnixNano()))

type Deck struct {
	Cards   [NUM_CARDS]Card
	current int
}

func NewDeck() Deck {
	var cards [NUM_CARDS]Card
	for c := 0; c < 4; c++ {
		for p := 0; p < 13; p++ {
			cards[c*13+p] = Card{cardPoint(p), cardColor(c)}
		}
	}
	cards[NUM_CARDS-2] = Card{BLACK_JOKER, NONE}
	cards[NUM_CARDS-1] = Card{RED_JOKER, NONE}
	return Deck{
		cards,
		NUM_CARDS - 1,
	}
}

func (d Deck) Size() int {
	return d.current + 1
}

func (d *Deck) Shuffle() {
	for i := d.current; i >= 0; i-- {
		idx := R.Intn(i + 1)
		d.Cards[i], d.Cards[idx] = d.Cards[idx], d.Cards[i]
	}
}

func (d *Deck) Deal(n int) []*Card {
	if n > d.Size() {
		panic(fmt.Sprintf("failed to deal %v card(s) from deck of size %v", n, d.Size()))
	}
	var cards []*Card
	for i := 0; i < n; i++ {
		cards = append(cards, &d.Cards[d.current])
		d.current--
	}
	return cards
}

func (d *Deck) String() string {
	var cards []string
	for i := 0; i < d.Size(); i++ {
		cards = append(cards, d.Cards[i].String())
	}
	return fmt.Sprintf("%v\n", cards)
}
