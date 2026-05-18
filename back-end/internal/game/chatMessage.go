package game

type ChatMessage struct {
	Sender  string `json:"sender"`
	Message string `json:"message"`
}

type ChatHistory struct {
	Messages []ChatMessage `json:"messages"`
}
