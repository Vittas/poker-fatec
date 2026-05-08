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

    // send existing chat history to the newly joined player
    history := room.GetChatHistory()
    player.Conn.WriteJSON(map[string]interface{}{
        "type": "CHAT_HISTORY",
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
            room.HandleAction(player, msg)
        }

        if msg.Type == "START" && msg.Start && len(room.Players) >= 2 {
            room.StartGame(msg)
        }

        if msg.Type == "CHAT" && msg.Message != "" {
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
    }


    conn.Close()
}