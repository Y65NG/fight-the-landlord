package util

import (
	"strings"
	"testing"
)

func TestCard(t *testing.T) {
	cards1 := []*Card {{Point: THREE}, {Point: FOUR}, {Point: THREE}, {Point: FOUR}, {Point: FOUR}}
	Sort(cards1)
	t.Log(cards1)
	cards2 := []*Card {{Point: TWO}, {Point: THREE}, {Point: THREE}, {Point: THREE}, {Point: TWO}}
	Sort(cards2)
	t.Log(cards2)
	t.Log(Valid(cards2))
	t.Log(CompareTo(cards1, cards2))
}

func TestSplit(t *testing.T) {
	str := "a, 2, 3"
	splittedStr := strings.Split(str, ", ")
	t.Log(splittedStr)
}
