package game

import (
    "sync"
)

type Room struct {
    ID         string
    Players    []*Player
    Deck       []Card
    Community  []Card
    Pot        int
    Turn       int
    Mutex      sync.Mutex
}

var Rooms = map[string]*Room{}

func CreateRoom(id string) *Room {
    room := &Room{
        ID:        id,
        Players:   []*Player{},
        Deck:      NewDeck(),
        Community: []Card{},
        Pot:       0,
        Turn:      0,
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

func (r *Room) Broadcast(data interface{}) {
    for _, player := range r.Players {
        player.Conn.WriteJSON(data)
    }
}

func (r *Room) StartGame() {
    if len(r.Players) < 2 {
        return
    }

    r.Deck = NewDeck()

    for _, player := range r.Players {
        player.Cards = []Card{
            r.DrawCard(),
            r.DrawCard(),
        }

        player.Conn.WriteJSON(map[string]interface{}{
            "type": "PRIVATE_HAND",
            "cards": player.Cards,
        })
    }

    r.Broadcast(map[string]interface{}{
        "type": "GAME_STARTED",
    })

    r.SendTurn()
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

func (r *Room) HandleAction(player *Player, msg Message) {
    r.Mutex.Lock()
    defer r.Mutex.Unlock()

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
