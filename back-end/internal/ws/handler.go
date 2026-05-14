package ws

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"poker-fatec/internal/game"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	roomID := strings.TrimPrefix(r.URL.Path, "/ws/")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	room := game.GetRoom(roomID)

	var joinMsg game.Message

	err = conn.ReadJSON(&joinMsg)
	if err != nil {
		conn.Close()
		return
	}

	player := &game.Player{
		Name:       joinMsg.Name,
		Conn:       conn,
		Chips:      1000,
		Folded:     false,
		LastActive: time.Now(),
	}

	room.AddPlayer(player)
	room.AssignPositions()

	room.Broadcast(map[string]interface{}{
		"type":     "PLAYER_JOINED",
		"position": player.Position,
		"player":   player.Name,
	})

	// send existing chat history to the newly joined player
	history := room.GetChatHistory()
	player.Conn.WriteJSON(map[string]interface{}{
		"type":     "CHAT_HISTORY",
		"messages": history.Messages,
	})

	if joinMsg.Start && len(room.Players) >= 2 {
		room.StartGame(joinMsg)
	}

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg game.Message

		err = json.Unmarshal(data, &msg)
		if err != nil {
			continue
		}

		if msg.Type == "ACTION" {
			player.LastActive = time.Now()
			room.HandleAction(player, msg)
		}

		if msg.Type == "START" && msg.Start && len(room.Players) >= 2 {
			player.LastActive = time.Now()
			room.StartGame(msg)
		}

		if msg.Type == "CHAT" && msg.Message != "" {
			player.LastActive = time.Now()
			chat := game.ChatMessage{
				Sender:  player.Name,
				Message: msg.Message,
			}

			room.AddChatMessage(chat)

			room.Broadcast(map[string]interface{}{
				"type":    "CHAT",
				"sender":  chat.Sender,
				"message": chat.Message,
			})
		}

		if msg.Type == "NEXT_ROUND" {
			if player.Position != "BTN" {
				continue
			}

			player.LastActive = time.Now()

			room.AdvanceRound()
		}

		if msg.Type == "KICK_INACTIVE" {
			if player.Position != "BTN" {
				continue
			}

			player.LastActive = time.Now()

			threshold := time.Duration(msg.InactiveSeconds) * time.Second
			if threshold <= 0 {
				threshold = 120 * time.Second
			}

			cutoff := time.Now().Add(-threshold)
			removed := room.RemoveInactivePlayers(cutoff)
			if len(removed) > 0 {
				room.Broadcast(map[string]interface{}{
					"type":    "PLAYERS_REMOVED",
					"players": removed,
				})
			}
		}

		if msg.Type == "TRANSFER" {
			player.LastActive = time.Now()
			fromChips, toChips, ok := room.TransferChips(player.Name, msg.Target, msg.Amount)
			if !ok {
				continue
			}

			room.Broadcast(map[string]interface{}{
				"type":   "CHIPS_TRANSFERRED",
				"from":   player.Name,
				"to":     msg.Target,
				"amount": msg.Amount,
				"fromChips": fromChips,
				"toChips":   toChips,
			})
		}
	}

	conn.Close()
}

func ListPlayersInRoom(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	roomID := strings.TrimPrefix(r.URL.Path, "/ws/players/")

	room := game.GetRoom(roomID)

	type PlayerInfo struct {
		Name     string `json:"name"`
		Position string `json:"position"`
	}

	var players []PlayerInfo

	for _, p := range room.Players {
		players = append(players, PlayerInfo{
			Name:     p.Name,
			Position: p.Position,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(players)
}

func GetPlayerPosition(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/ws/player/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	roomID := parts[0]
	name, err := url.PathUnescape(parts[1])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	room := game.GetRoom(roomID)

	for _, p := range room.Players {
		if p.Name == name {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"name":     p.Name,
				"position": p.Position,
			})
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
}

func GetPlayerChips(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/ws/player/")
	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 3 || parts[0] == "" || parts[1] == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if parts[2] != "chips" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	roomID := parts[0]
	name, err := url.PathUnescape(parts[1])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	room := game.GetRoom(roomID)

	for _, p := range room.Players {
		if p.Name == name {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"name":  p.Name,
				"chips": p.Chips,
			})
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
}