package util

import (
	"fmt"

	"golang.org/x/exp/slices"
)

type Card struct {
	Point cardPoint
	Color cardColor
}

func (c Card) String() string {
	var color string
	var point string
	switch c.Color {
	case SPADE:
		color = "♠"
	case CLUBS:
		color = "♣"
	case HEART:
		color = "♥"
	case DIAMOND:
		color = "♦"
	case NONE:
		color = ""
	}
	switch c.Point {
	case THREE:
		point = " 3"
	case FOUR:
		point = " 4"
	case FIVE:
		point = " 5"
	case SIX:
		point = " 6"
	case SEVEN:
		point = " 7"
	case EIGHT:
		point = " 8"
	case NINE:
		point = " 9"
	case TEN:
		point = " 10"
	case JACK:
		point = " J"
	case QUEEN:
		point = " Q"
	case KING:
		point = " K"
	case ACE:
		point = " A"
	case TWO:
		point = " 2"
	case BLACK_JOKER:
		point = "joker"
	case RED_JOKER:
		point = "JOKER"
	}
	return fmt.Sprintf("%v%v", color, point)
}

func CardsToString(cards []*Card) string {
	str := "["
	for _, c := range cards {
		str += c.String() + " "
	}
	if len(str) > 1 {
		str = str[:len(str)-1]
	}
	str += "]"
	return str
}

func (c Card) Equal(c2 Card) bool {
	return c.Point == c2.Point
}

type cardColor int

const (
	SPADE cardColor = iota
	CLUBS
	HEART
	DIAMOND
	NONE
)

type cardPoint int

const (
	THREE cardPoint = iota
	FOUR
	FIVE
	SIX
	SEVEN
	EIGHT
	NINE
	TEN
	JACK
	QUEEN
	KING
	ACE
	TWO
	BLACK_JOKER
	RED_JOKER
)

func Valid(cards []*Card) bool {
	if len(cards) == 0 {
		return true
	}
	if isSingle(cards) || isDouble(cards) || isBomb(cards) || isStraight(cards) || isDoubleStraight(cards) || isTriple(cards) || isTripleWithOne(cards) || isTripleWithTwo(cards) || isPlane(cards) {
		return true
	}
	return false
}

func Contains(cards []*Card, cardsInfo []*Card) bool {
	if len(cardsInfo) == 0 {
		return true
	}
	if len(cards) < len(cardsInfo) {
		return false
	}
	visitedCardsIdx := make(map[int]struct{})
	for _, cardInfo := range cardsInfo {
		notFound := true
		for i, c := range cards {
			if _, ok := visitedCardsIdx[i]; !ok && cardInfo.Equal(*c) {
				notFound = false
				visitedCardsIdx[i] = struct{}{}
				break
			}
		}
		if notFound {
			return false
		}
	}
	return true
}

func Sort(cards []*Card) {
	slices.SortFunc(cards, func(c1, c2 *Card) int {
		switch {
		case c1.Point < c2.Point:
			return -1
		case c1.Point == c2.Point:
			return 0
		default:
			return 1
		}
	})
	switch {
	case isSingle(cards) || isDouble(cards) || isBomb(cards) || isStraight(cards) || isDoubleStraight(cards) || isTriple(cards):
		return
	case len(cards) == 4:
		for i := 0; i < len(cards)-2; i++ {
			if cards[i].Point == cards[i+1].Point && cards[i].Point == cards[i+2].Point {
				cards[0], cards[i] = cards[i], cards[0]
				cards[1], cards[i+1] = cards[i+1], cards[1]
				cards[2], cards[i+2] = cards[i+2], cards[2]
				break
			}
		}
	case len(cards) == 5:
		for i := 0; i < len(cards)-2; i++ {
			if cards[i].Point == cards[i+1].Point && cards[i].Point == cards[i+2].Point {
				cards[0], cards[i] = cards[i], cards[0]
				cards[1], cards[i+1] = cards[i+1], cards[1]
				cards[2], cards[i+2] = cards[i+2], cards[2]
				break
			}
		}
	case len(cards) == 6 || len(cards) == 8 || len(cards) == 10:
		for i := 0; i < len(cards)-2; i++ {
			if cards[i].Point == cards[i+1].Point && cards[i].Point == cards[i+2].Point {
				cards[0], cards[i] = cards[i], cards[0]
				cards[1], cards[i+1] = cards[i+1], cards[1]
				cards[2], cards[i+2] = cards[i+2], cards[2]
				break
			}
		}
		for i := 3; i < len(cards)-2; i++ {
			if cards[i].Point == cards[i+1].Point && cards[i].Point == cards[i+2].Point {
				cards[3], cards[i] = cards[i], cards[3]
				cards[4], cards[i+1] = cards[i+1], cards[4]
				cards[5], cards[i+2] = cards[i+2], cards[5]
				break
			}
		}
	}
}

func CompareTo(cards, lastCards []*Card) bool {
	// cards and lastCards are sorted
	if !Valid(cards) || !Valid(lastCards) {
		return false
	}
	if len(lastCards) == 0 {
		return len(cards) != 0
	}
	if len(cards) == 0 {
		return true
	}

	switch {
	case isBomb(cards):
		if isBomb(lastCards) && lastCards[0].Point > cards[0].Point {
			return false
		}
	case len(cards) == len(lastCards):
		switch {
		case isSingle(cards), isTriple(cards):
			if cards[0].Point <= lastCards[0].Point {
				return false
			}
		case isDouble(cards):
			if !isDouble(cards) || cards[0].Point <= lastCards[0].Point {
				return false
			}
		case isTripleWithOne(cards):
			if !isTripleWithOne(cards) || cards[0].Point <= lastCards[0].Point {
				return false
			}
		case isTripleWithTwo(cards):
			if !isTripleWithTwo(cards) || cards[0].Point <= lastCards[0].Point {
				return false
			}
		case isStraight(cards):
			if !isStraight(cards) || cards[0].Point <= lastCards[0].Point {
				return false
			}
		case isDoubleStraight(cards):
			if !isDoubleStraight(lastCards) || cards[0].Point <= lastCards[0].Point {
				return false
			}
		case isPlane(cards):
			if !isPlane(lastCards) || cards[0].Point <= lastCards[0].Point {
				return false
			}
		}
	default:
		return false
	}
	return true
}

func isBomb(cards []*Card) bool {
	if len(cards) != 4 {
		if len(cards) == 2 {
			return cards[0].Point == BLACK_JOKER && cards[1].Point == RED_JOKER
		}
		return false
	}
	for i := 0; i < len(cards)-1; i++ {
		if cards[i].Point != cards[i+1].Point {
			return false
		}
	}
	return true
}

func isSingle(cards []*Card) bool {
	return len(cards) == 1
}

func isDouble(cards []*Card) bool {
	if len(cards) != 2 {
		return false
	}
	if cards[0].Point != cards[1].Point {
		return false
	}
	return true
}

func isTriple(cards []*Card) bool {
	if len(cards) != 3 {
		return false
	}
	if cards[0].Point != cards[1].Point || cards[0].Point != cards[2].Point {
		return false
	}
	return true
}

func isTripleWithOne(cards []*Card) bool {
	if len(cards) != 4 {
		return false
	}
	if cards[0].Point != cards[1].Point || cards[0].Point != cards[2].Point {
		return false
	}
	return true
}

func isTripleWithTwo(cards []*Card) bool {
	if len(cards) != 5 {
		return false
	}
	if cards[0].Point != cards[1].Point || cards[0].Point != cards[2].Point {
		return false
	}
	if cards[3].Point != cards[4].Point {
		return false
	}
	return true
}

func isStraight(cards []*Card) bool {
	if len(cards) < 5 {
		return false
	}
	for i := 0; i < len(cards)-1; i++ {
		if cards[i].Point != cards[i+1].Point-1 {
			return false
		}
	}
	return true
}

func isDoubleStraight(cards []*Card) bool {
	if len(cards) < 6 {
		return false
	}
	if len(cards)%2 != 0 {
		return false
	}
	for i := 0; i < len(cards)-2; i += 2 {
		if cards[i].Point != cards[i+1].Point {
			return false
		}
		if cards[i].Point != cards[i+2].Point-1 {
			return false
		}
	}
	return true
}

func isPlane(cards []*Card) bool {
	if len(cards) != 6 && len(cards) != 8 && len(cards) != 10 {
		return false
	}
	if cards[0].Point != cards[3].Point-1 {
		return false
	}

	for i := 0; i < 6; i += 3 {
		if cards[i].Point != cards[i+1].Point || cards[i].Point != cards[i+2].Point {
			return false
		}
	}
	if len(cards) == 10 {
		if cards[6].Point != cards[7].Point || cards[8].Point != cards[9].Point {
			return false
		}
	}
	return true
}
