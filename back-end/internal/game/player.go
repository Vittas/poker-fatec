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
	Position   string
	LastActive time.Time
}
