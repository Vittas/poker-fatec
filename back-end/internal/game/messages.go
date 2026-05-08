package game

type Message struct {
    Type   string `json:"type"`
    Name   string `json:"name,omitempty"`
    Action string `json:"action,omitempty"`
    Amount int    `json:"amount,omitempty"`
    Start  bool   `json:"start,omitempty"`
    Message string `json:"message,omitempty"`
}