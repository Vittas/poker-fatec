package ws

import (
    "encoding/json"
    "net/http"
    "strings"

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
        Name:   joinMsg.Name,
        Conn:   conn,
        Chips:  1000,
        Folded: false,
    }

    room.AddPlayer(player)

    room.Broadcast(map[string]interface{}{
        "type":  "PLAYER_JOINED",
        "player": player.Name,
    })

    if len(room.Players) >= 2 {
        room.StartGame()
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
            room.HandleAction(player, msg)
        }
    }

    conn.Close()
}