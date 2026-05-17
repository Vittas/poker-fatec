package game

import (
	"time"

	"github.com/gorilla/websocket"
)

type Player struct {
	Name       string
	Conn       *websocket.Conn
	Chips      int
	Cards      []Card
	Folded     bool
	AllIn      bool
	BetInRound int
	TotalBet   int
	Position   string
	LastActive time.Time
}
