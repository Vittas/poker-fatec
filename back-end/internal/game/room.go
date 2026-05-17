package game

import (
	"sync"
	"time"
)

type Room struct {
	ID             string
	Players        []*Player
	Start          bool
	Deck           []Card
	Community      []Card
	Pot            int
	Round          int
	DealerIndex    int
	Turn           int
	SmallBlind     int
	BigBlind       int
	CurrentBet     int
	MinRaise       int
	PendingActions int
	Mutex          sync.Mutex
	History        ChatHistory
}

type SidePot struct {
	Amount   int
	Eligible []*Player
}

var Rooms = map[string]*Room{}

func CreateRoom(id string) *Room {
	room := &Room{
		ID:             id,
		Players:        []*Player{},
		Deck:           NewDeck(),
		Community:      []Card{},
		Pot:            0,
		Round:          0,
		DealerIndex:    0,
		Turn:           0,
		SmallBlind:     100,
		BigBlind:       200,
		CurrentBet:     0,
		MinRaise:       0,
		PendingActions: 0,
		History:        ChatHistory{},
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
	if r.Round > 0 && r.activePlayersCount() <= 1 {
		r.finishHand()
	}

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

	if len(r.Players) == 2 {
		r.Players[r.DealerIndex].Position = "SB"
		r.Players[(r.DealerIndex+1)%len(r.Players)].Position = "BB"
		return
	}

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
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	if len(r.Players) < 2 && !msg.Start {
		return
	}

	if r.SmallBlind <= 0 {
		r.SmallBlind = 100
	}
	if r.BigBlind <= 0 {
		r.BigBlind = r.SmallBlind * 2
	}

	r.Deck = NewDeck()
	r.Round = 1
	r.Turn = 0
	r.Community = []Card{}
	r.Pot = 0
	r.CurrentBet = 0
	r.MinRaise = r.BigBlind
	r.PendingActions = 0
	r.AssignPositions()

	for _, player := range r.Players {
		player.Folded = false
		player.AllIn = false
		player.BetInRound = 0
		player.TotalBet = 0
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

	r.postBlindsLocked()

	r.Broadcast(map[string]interface{}{
		"type": "GAME_STARTED",
	})

	r.startBettingRoundLocked(true)
	r.sendTurnLocked()
}

func (r *Room) AdvanceRound() {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	r.advanceRoundLocked()
}

func (r *Room) advanceRoundLocked() {

	if r.Round >= 5 {
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
		return
	}

	r.Broadcast(map[string]interface{}{
		"type":      "ROUND_ADVANCED",
		"round":     r.Round,
		"community": r.Community,
	})

	r.startBettingRoundLocked(false)
	r.sendTurnLocked()
}

func (r *Room) finishHand() {
	if r.Pot <= 0 {
		return
	}

	awards := r.distributePotsLocked()
	if len(awards) == 0 {
		return
	}

	potAwarded := r.Pot
	r.Pot = 0

	r.Broadcast(map[string]interface{}{
		"type":   "HAND_FINISHED",
		"winner": awards[0]["winner"],
		"pot":    potAwarded,
		"pots":   awards,
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
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	r.sendTurnLocked()
}

func (r *Room) sendTurnLocked() {
	if len(r.Players) == 0 {
		return
	}

	if r.activePlayersCount() <= 1 {
		r.finishHand()
		return
	}

	if r.Players[r.Turn].Folded || r.Players[r.Turn].AllIn {
		nextIndex, ok := r.nextActionableIndex(r.Turn)
		if !ok {
			return
		}
		r.Turn = nextIndex
	}

	currentPlayer := r.Players[r.Turn]

	r.Broadcast(map[string]interface{}{
		"type":   "TURN",
		"player": currentPlayer.Name,
	})
}

func (r *Room) NextTurn() {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	r.nextTurnLocked()
}

func (r *Room) nextTurnLocked() {
	if r.activePlayersCount() <= 1 {
		r.finishHand()
		return
	}

	nextIndex, ok := r.nextActionableIndex(r.Turn)
	if !ok {
		return
	}

	r.Turn = nextIndex
	r.sendTurnLocked()
}

func (r *Room) nextActionableIndex(from int) (int, bool) {
	if len(r.Players) == 0 {
		return 0, false
	}

	for i := 1; i <= len(r.Players); i++ {
		idx := (from + i) % len(r.Players)
		if !r.Players[idx].Folded && !r.Players[idx].AllIn {
			return idx, true
		}
	}

	return 0, false
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
	if player.Folded || player.AllIn {
		return
	}

	toCall := r.CurrentBet - player.BetInRound
	if toCall < 0 {
		toCall = 0
	}

	acted := false
	raised := false

	switch msg.Action {
	case "FOLD":
		player.Folded = true
		acted = true

	case "CHECK":
		if toCall == 0 {
			acted = true
		}

	case "CALL":
		if toCall == 0 {
			acted = true
			break
		}

		callAmount := r.minInt(player.Chips, toCall)
		r.recordBetLocked(player, callAmount)
		if callAmount < toCall {
			player.AllIn = true
		}
		acted = true

	case "BET":
		if r.CurrentBet != 0 {
			break
		}
		betAmount := r.minInt(player.Chips, msg.Amount)
		if betAmount <= 0 {
			break
		}
		r.recordBetLocked(player, betAmount)
		if betAmount < msg.Amount || player.Chips == 0 {
			player.AllIn = player.Chips == 0
		}
		r.CurrentBet = player.BetInRound
		r.MinRaise = betAmount
		raised = true
		acted = true

	case "RAISE":
		if r.CurrentBet == 0 {
			break
		}
		raiseBy := msg.Amount
		if raiseBy < r.MinRaise && player.Chips > toCall+raiseBy {
			break
		}
		raiseAmount := r.minInt(player.Chips, toCall+raiseBy)
		r.recordBetLocked(player, raiseAmount)
		if raiseAmount < toCall+raiseBy {
			player.AllIn = true
		}
		if player.BetInRound > r.CurrentBet {
			r.CurrentBet = player.BetInRound
			r.MinRaise = raiseBy
			raised = true
		}
		acted = true

	case "ALL_IN":
		allInAmount := player.Chips
		if allInAmount <= 0 {
			break
		}
		r.recordBetLocked(player, allInAmount)
		player.AllIn = true
		if player.BetInRound > r.CurrentBet {
			raiseBy := player.BetInRound - r.CurrentBet
			if raiseBy >= r.MinRaise {
				r.CurrentBet = player.BetInRound
				r.MinRaise = raiseBy
				raised = true
			}
		}
		acted = true
	}

	if !acted {
		return
	}

	r.Broadcast(map[string]interface{}{
		"type":   "PLAYER_ACTION",
		"player": player.Name,
		"action": msg.Action,
		"amount": msg.Amount,
		"pot":    r.Pot,
	})

	if r.activePlayersCount() <= 1 {
		r.finishHand()
		return
	}

	if raised {
		r.PendingActions = r.actionablePlayersCount() - 1
	} else {
		r.PendingActions--
	}

	if r.PendingActions <= 0 {
		r.advanceRoundLocked()
		return
	}

	r.nextTurnLocked()
}

func (r *Room) activePlayersCount() int {
	count := 0
	for _, player := range r.Players {
		if !player.Folded {
			count++
		}
	}

	return count
}

func (r *Room) actionablePlayersCount() int {
	count := 0
	for _, player := range r.Players {
		if !player.Folded && !player.AllIn {
			count++
		}
	}

	return count
}

func (r *Room) postBlindsLocked() {
	if len(r.Players) == 0 {
		return
	}

	sbIndex, bbIndex := r.blindIndexesLocked()
	if sbIndex >= 0 {
		r.recordBetLocked(r.Players[sbIndex], r.minInt(r.Players[sbIndex].Chips, r.SmallBlind))
		if r.Players[sbIndex].Chips == 0 {
			r.Players[sbIndex].AllIn = true
		}
	}
	if bbIndex >= 0 {
		r.recordBetLocked(r.Players[bbIndex], r.minInt(r.Players[bbIndex].Chips, r.BigBlind))
		if r.Players[bbIndex].Chips == 0 {
			r.Players[bbIndex].AllIn = true
		}
	}

	r.CurrentBet = 0
	if sbIndex >= 0 && r.Players[sbIndex].BetInRound > r.CurrentBet {
		r.CurrentBet = r.Players[sbIndex].BetInRound
	}
	if bbIndex >= 0 && r.Players[bbIndex].BetInRound > r.CurrentBet {
		r.CurrentBet = r.Players[bbIndex].BetInRound
	}
	r.MinRaise = r.BigBlind
}

func (r *Room) blindIndexesLocked() (int, int) {
	if len(r.Players) == 0 {
		return -1, -1
	}
	if len(r.Players) == 2 {
		return r.DealerIndex, (r.DealerIndex + 1) % len(r.Players)
	}
	return (r.DealerIndex + 1) % len(r.Players), (r.DealerIndex + 2) % len(r.Players)
}

func (r *Room) startBettingRoundLocked(preflop bool) {
	if !preflop {
		for _, player := range r.Players {
			player.BetInRound = 0
		}
		r.CurrentBet = 0
		r.MinRaise = r.BigBlind
	}

	r.PendingActions = r.actionablePlayersCount()

	if r.PendingActions == 0 {
		r.advanceToShowdownLocked()
		return
	}

	if preflop {
		_, bbIndex := r.blindIndexesLocked()
		r.Turn = r.firstActionableIndex(bbIndex)
		return
	}

	r.Turn = r.firstActionableIndex(r.DealerIndex)
}

func (r *Room) firstActionableIndex(from int) int {
	idx, ok := r.nextActionableIndex(from)
	if !ok {
		return from
	}
	return idx
}

func (r *Room) advanceToShowdownLocked() {
	for r.Round < 5 {
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
			return
		}

		r.Broadcast(map[string]interface{}{
			"type":      "ROUND_ADVANCED",
			"round":     r.Round,
			"community": r.Community,
		})
	}
}

func (r *Room) recordBetLocked(player *Player, amount int) {
	if amount <= 0 {
		return
	}

	if amount > player.Chips {
		amount = player.Chips
	}

	player.Chips -= amount
	player.BetInRound += amount
	player.TotalBet += amount
	r.Pot += amount
}

func (r *Room) distributePotsLocked() []map[string]interface{} {
	pots := r.buildSidePotsLocked()
	awards := []map[string]interface{}{}

	for _, pot := range pots {
		winner := r.findWinnerAmong(pot.Eligible)
		if winner == nil {
			continue
		}
		winner.Chips += pot.Amount
		awards = append(awards, map[string]interface{}{
			"winner": winner.Name,
			"amount": pot.Amount,
		})
	}

	return awards
}

func (r *Room) buildSidePotsLocked() []SidePot {
	levels := []int{}
	for _, player := range r.Players {
		if player.TotalBet > 0 {
			levels = append(levels, player.TotalBet)
		}
	}

	if len(levels) == 0 {
		return nil
	}

	levels = r.uniqueSorted(levels)

	prev := 0
	pots := []SidePot{}
	for _, level := range levels {
		count := 0
		eligible := []*Player{}
		for _, player := range r.Players {
			if player.TotalBet >= level {
				count++
			}
			if !player.Folded && player.TotalBet >= level {
				eligible = append(eligible, player)
			}
		}

		amount := (level - prev) * count
		if amount > 0 {
			pots = append(pots, SidePot{Amount: amount, Eligible: eligible})
		}
		prev = level
	}

	return pots
}

func (r *Room) findWinnerAmong(players []*Player) *Player {
	for _, player := range players {
		if !player.Folded {
			return player
		}
	}

	return nil
}

func (r *Room) uniqueSorted(values []int) []int {
	seen := map[int]bool{}
	unique := []int{}
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			unique = append(unique, value)
		}
	}

	for i := 0; i < len(unique); i++ {
		for j := i + 1; j < len(unique); j++ {
			if unique[j] < unique[i] {
				unique[i], unique[j] = unique[j], unique[i]
			}
		}
	}

	return unique
}

func (r *Room) minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func (r *Room) GetChatHistory() ChatHistory {
	return r.History
}

func (r *Room) AddChatMessage(msg ChatMessage) ChatHistory {
	r.History.Messages = append(r.History.Messages, msg)
	return r.History
}
