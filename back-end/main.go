package main

import (
    "fmt"
    "net/http"

    "poker-fatec/internal/ws"
)

func main() {
    http.HandleFunc("/ws/", ws.HandleWebSocket)

    fmt.Println("Servidor iniciado na porta 8080")

    err := http.ListenAndServe(":8080", nil)
    if err != nil {
        panic(err)
    }
}