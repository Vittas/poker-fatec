package game

import (
	"sync"
	"time"
)

type Room struct {
	ID        string
	Players   []*Player
	Start     bool
	Deck      []Card
	Community []Card
	Pot       int
	Round     int
	DealerIndex int
	Turn      int
	Mutex     sync.Mutex
	History   ChatHistory
}

var Rooms = map[string]*Room{}

func CreateRoom(id string) *Room {
	room := &Room{
		ID:        id,
		Players:   []*Player{},
		Deck:      NewDeck(),
		Community: []Card{},
		Pot:       0,
		Round:     0,
		DealerIndex: 0,
		Turn:      0,
		History:   ChatHistory{},
	}

	Rooms[id] = room

	return room
}

func GetRoom(id string) *Room {
	room, exists := Rooms[id]

	if !exists {
		return CreateRoom(id)
	}

	return room
}

func (r *Room) AddPlayer(player *Player) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	r.Players = append(r.Players, player)
}

func (r *Room) RemoveInactivePlayers(cutoff time.Time) []string {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	removed := []string{}
	active := []*Player{}

	for _, player := range r.Players {
		if player.LastActive.Before(cutoff) {
			removed = append(removed, player.Name)
			player.Conn.Close()
			continue
		}

		active = append(active, player)
	}

	r.Players = active
	r.AssignPositions()

	return removed
}

func (r *Room) AssignPositions() {
	if len(r.Players) == 0 {
		return
	}

	for _, player := range r.Players {
		player.Position = "OUTHER"
	}

	if r.DealerIndex < 0 || r.DealerIndex >= len(r.Players) {
		r.DealerIndex = 0
	}

	r.Players[r.DealerIndex].Position = "BTN"

	if len(r.Players) >= 2 {
		r.Players[(r.DealerIndex+1)%len(r.Players)].Position = "SB"
	}

	if len(r.Players) >= 3 {
		r.Players[(r.DealerIndex+2)%len(r.Players)].Position = "BB"
	}
}

func (r *Room) Broadcast(data interface{}) {
	for _, player := range r.Players {
		player.Conn.WriteJSON(data)
	}
}

func (r *Room) StartGame(msg Message) {
	if len(r.Players) < 2 && !msg.Start {
		return
	}

	r.Deck = NewDeck()
	r.Round = 1
	r.Turn = 0
	r.Community = []Card{}
	r.Pot = 0

	for _, player := range r.Players {
		player.Folded = false
	}

	for _, player := range r.Players {
		player.Cards = []Card{
			r.DrawCard(),
			r.DrawCard(),
		}

		player.Conn.WriteJSON(map[string]interface{}{
			"type":  "PRIVATE_HAND",
			"cards": player.Cards,
		})
	}

	r.Broadcast(map[string]interface{}{
		"type": "GAME_STARTED",
	})

	r.SendTurn()
}

func (r *Room) AdvanceRound() {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	if r.Round >= 4 {
		return
	}

	r.Round++

	switch r.Round {
	case 2:
		r.Community = append(r.Community, r.DrawCard(), r.DrawCard(), r.DrawCard())

	case 3:
		r.Community = append(r.Community, r.DrawCard())

	case 4:
		r.Community = append(r.Community, r.DrawCard())
	case 5:
		r.finishHand()
	}

	r.Broadcast(map[string]interface{}{
		"type":      "ROUND_ADVANCED",
		"round":     r.Round,
		"community": r.Community,
	})
}

func (r *Room) finishHand() {
	winner := r.findWinner()
	if winner == nil {
		return
	}

	potAwarded := r.Pot
	winner.Chips += potAwarded
	r.Pot = 0

	r.Broadcast(map[string]interface{}{
		"type":   "HAND_FINISHED",
		"winner": winner.Name,
		"pot":    potAwarded,
	})

	r.rotateDealer()
}

func (r *Room) rotateDealer() {
	if len(r.Players) == 0 {
		return
	}

	r.DealerIndex = (r.DealerIndex + 1) % len(r.Players)
	r.AssignPositions()

	r.Broadcast(map[string]interface{}{
		"type": "POSITIONS_UPDATED",
	})
}

func (r *Room) findWinner() *Player {
	for _, player := range r.Players {
		if !player.Folded {
			return player
		}
	}

	return nil
}

func (r *Room) DrawCard() Card {
	card := r.Deck[0]
	r.Deck = r.Deck[1:]

	return card
}

func (r *Room) SendTurn() {
	currentPlayer := r.Players[r.Turn]

	r.Broadcast(map[string]interface{}{
		"type":   "TURN",
		"player": currentPlayer.Name,
	})
}

func (r *Room) NextTurn() {
	r.Turn++

	if r.Turn >= len(r.Players) {
		r.Turn = 0
	}

	r.SendTurn()
}

func (r *Room) TransferChips(fromName string, toName string, amount int) (int, int, bool) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	if amount <= 0 {
		return 0, 0, false
	}

	var fromPlayer *Player
	var toPlayer *Player

	for _, player := range r.Players {
		if player.Name == fromName {
			fromPlayer = player
		}
		if player.Name == toName {
			toPlayer = player
		}
	}

	if fromPlayer == nil || toPlayer == nil {
		return 0, 0, false
	}

	if fromPlayer.Chips < amount {
		return 0, 0, false
	}

	fromPlayer.Chips -= amount
	toPlayer.Chips += amount

	return fromPlayer.Chips, toPlayer.Chips, true
}

func (r *Room) HandleAction(player *Player, msg Message) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	if len(r.Players) < 2 {
		return
	}

	if len(r.Players) == 0 || r.Players[r.Turn] != player {
		return
	}

	switch msg.Action {
	case "FOLD":
		player.Folded = true

	case "CALL":
		r.Pot += 10
		player.Chips -= 10

	case "RAISE":
		r.Pot += msg.Amount
		player.Chips -= msg.Amount
	}

	r.Broadcast(map[string]interface{}{
		"type":   "PLAYER_ACTION",
		"player": player.Name,
		"action": msg.Action,
		"amount": msg.Amount,
		"pot":    r.Pot,
	})

	r.NextTurn()
}

func (r *Room) GetChatHistory() ChatHistory {
	return r.History
}

func (r *Room) AddChatMessage(msg ChatMessage) ChatHistory {
	r.History.Messages = append(r.History.Messages, msg)
	return r.History
}
