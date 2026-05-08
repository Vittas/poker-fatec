package game

import (
    "math/rand"
    "time"
)

func NewDeck() []Card {
    suits := []string{"espada", "copa", "ouro", "paus"}
    ranks := []string{
        "2", "3", "4", "5", "6", "7", "8", "9", "10",
        "J", "Q", "K", "A", 
    }

    deck := []Card{}

    for _, suit := range suits {
        for _, rank := range ranks {
            deck = append(deck, Card{
                Suit: suit,
                Rank: rank,
            })
        }
    }

    rand.Seed(time.Now().UnixNano())

    rand.Shuffle(len(deck), func(i, j int) {
        deck[i], deck[j] = deck[j], deck[i]
    })

    return deck
}