package game

import (
	"sort"
)

type HandRank int

const (
	HighCard HandRank = iota
	Pair
	TwoPair
	ThreeOfAKind
	Straight
	Flush
	FullHouse
	FourOfAKind
	StraightFlush
	RoyalFlush
)

type Hand struct {
	Rank  HandRank
	Cards []Card
}

func EvaluateHand(cards []Card) Hand {
	sort.Slice(cards, func(i, j int) bool {
		return rankValue(cards[i].Rank) < rankValue(cards[j].Rank)
	})

	if isRoyalFlush(cards) {
		return Hand{Rank: RoyalFlush, Cards: cards}
	} else if isStraightFlush(cards) {
		return Hand{Rank: StraightFlush, Cards: cards}
	} else if isFourOfAKind(cards) {
		return Hand{Rank: FourOfAKind, Cards: cards}
	} else if isFullHouse(cards) {
		return Hand{Rank: FullHouse, Cards: cards}
	} else if isFlush(cards) {
		return Hand{Rank: Flush, Cards: cards}
	} else if isStraight(cards) {
		return Hand{Rank: Straight, Cards: cards}
	} else if isThreeOfAKind(cards) {
		return Hand{Rank: ThreeOfAKind, Cards: cards}
	} else if isTwoPair(cards) {
		return Hand{Rank: TwoPair, Cards: cards}
	} else if isPair(cards) {
		return Hand{Rank: Pair, Cards: cards}
	} else {
		return Hand{Rank: HighCard, Cards: cards}
	}
}

func EvaluateBestHand(cards []Card) Hand {
	if len(cards) <= 5 {
		copyCards := append([]Card(nil), cards...)
		return EvaluateHand(copyCards)
	}

	best := Hand{Rank: HighCard}
	first := true
	for i := 0; i < len(cards)-4; i++ {
		for j := i + 1; j < len(cards)-3; j++ {
			for k := j + 1; k < len(cards)-2; k++ {
				for l := k + 1; l < len(cards)-1; l++ {
					for m := l + 1; m < len(cards); m++ {
						combo := []Card{cards[i], cards[j], cards[k], cards[l], cards[m]}
						hand := EvaluateHand(append([]Card(nil), combo...))
						if first || CompareHands(hand, best) > 0 {
							best = hand
							first = false
						}
					}
				}
			}
		}
	}

	return best
}

func CompareHands(a Hand, b Hand) int {
	if a.Rank != b.Rank {
		if a.Rank > b.Rank {
			return 1
		}
		return -1
	}

	aRank, aTie := handScore(a.Cards, a.Rank)
	_, bTie := handScore(b.Cards, b.Rank)
	if aRank != b.Rank {
		if aRank > b.Rank {
			return 1
		}
		return -1
	}

	maxLen := len(aTie)
	if len(bTie) > maxLen {
		maxLen = len(bTie)
	}
	for i := 0; i < maxLen; i++ {
		var aVal int
		var bVal int
		if i < len(aTie) {
			aVal = aTie[i]
		}
		if i < len(bTie) {
			bVal = bTie[i]
		}
		if aVal != bVal {
			if aVal > bVal {
				return 1
			}
			return -1
		}
	}

	return 0
}

func isRoyalFlush(cards []Card) bool {
	if !isStraightFlush(cards) {
		return false
	}
	values := rankValues(cards)
	return values[0] == 10 && values[4] == 14
}

func isStraightFlush(cards []Card) bool {
	return isFlush(cards) && isStraight(cards)
}

func isFourOfAKind(cards []Card) bool {
	return (cards[0].Rank == cards[3].Rank) || (cards[1].Rank == cards[4].Rank)
}

func isFullHouse(cards []Card) bool {
	return (cards[0].Rank == cards[2].Rank && cards[3].Rank == cards[4].Rank) ||
		(cards[0].Rank == cards[1].Rank && cards[2].Rank == cards[4].Rank)
}

func isFlush(cards []Card) bool {
	for i := 1; i < len(cards); i++ {
		if cards[i].Suit != cards[0].Suit {
			return false
		}
	}
	return true
}

func isStraight(cards []Card) bool {
	values := rankValues(cards)
	for i := 1; i < len(values); i++ {
		if values[i] != values[i-1]+1 {
			return isWheelStraight(values)
		}
	}
	return true
}

func isThreeOfAKind(cards []Card) bool {
	return (cards[0].Rank == cards[2].Rank) || (cards[1].Rank == cards[3].Rank) || (cards[2].Rank == cards[4].Rank)
}

func isTwoPair(cards []Card) bool {
	return (cards[0].Rank == cards[1].Rank && cards[2].Rank == cards[3].Rank) ||
		(cards[0].Rank == cards[1].Rank && cards[3].Rank == cards[4].Rank) ||
		(cards[1].Rank == cards[2].Rank && cards[3].Rank == cards[4].Rank)
}

func isPair(cards []Card) bool {
	for i := 0; i < len(cards)-1; i++ {
		if cards[i].Rank == cards[i+1].Rank {
			return true
		}
	}
	return false
}

func rankValue(rank string) int {
	switch rank {
	case "2":
		return 2
	case "3":
		return 3
	case "4":
		return 4
	case "5":
		return 5
	case "6":
		return 6
	case "7":
		return 7
	case "8":
		return 8
	case "9":
		return 9
	case "10":
		return 10
	case "J":
		return 11
	case "Q":
		return 12
	case "K":
		return 13
	case "A":
		return 14
	default:
		return 0
	}
}

func rankValues(cards []Card) []int {
	values := make([]int, len(cards))
	for i, card := range cards {
		values[i] = rankValue(card.Rank)
	}
	sort.Ints(values)
	return values
}

func isWheelStraight(values []int) bool {
	if len(values) != 5 {
		return false
	}
	return values[0] == 2 && values[1] == 3 && values[2] == 4 && values[3] == 5 && values[4] == 14
}

func handScore(cards []Card, rank HandRank) (HandRank, []int) {
	valuesAsc := rankValues(cards)
	valuesDesc := reverseInts(valuesAsc)
	counts := valueCounts(valuesAsc)

	switch rank {
	case RoyalFlush:
		return rank, []int{14}
	case StraightFlush, Straight:
		return rank, []int{straightHigh(valuesAsc)}
	case FourOfAKind:
		quads := valuesByCount(counts, 4)
		kickers := valuesByCount(counts, 1)
		return rank, append(quads, kickers...)
	case FullHouse:
		trips := valuesByCount(counts, 3)
		pairs := valuesByCount(counts, 2)
		return rank, append(trips, pairs...)
	case Flush, HighCard:
		return rank, valuesDesc
	case ThreeOfAKind:
		trips := valuesByCount(counts, 3)
		kickers := valuesByCount(counts, 1)
		return rank, append(trips, kickers...)
	case TwoPair:
		pairs := valuesByCount(counts, 2)
		kickers := valuesByCount(counts, 1)
		return rank, append(pairs, kickers...)
	case Pair:
		pairs := valuesByCount(counts, 2)
		kickers := valuesByCount(counts, 1)
		return rank, append(pairs, kickers...)
	default:
		return rank, valuesDesc
	}
}

func valueCounts(values []int) map[int]int {
	counts := map[int]int{}
	for _, value := range values {
		counts[value]++
	}
	return counts
}

func valuesByCount(counts map[int]int, count int) []int {
	values := []int{}
	for value, c := range counts {
		if c == count {
			values = append(values, value)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(values)))
	return values
}

func reverseInts(values []int) []int {
	result := make([]int, len(values))
	for i := 0; i < len(values); i++ {
		result[i] = values[len(values)-1-i]
	}
	return result
}

func straightHigh(values []int) int {
	if isWheelStraight(values) {
		return 5
	}
	return values[len(values)-1]
}
