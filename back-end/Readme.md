Poker Fatec API

Base URL
- http://localhost:8080

HTTP Endpoints

1) List players in a room
- Method: GET
- Path: /ws/players/{roomId}
- Response 200 (application/json):
  [
    {"name":"Alice","position":"BTN"},
    {"name":"Bob","position":"SB"}
  ]

2) Get a player's position
- Method: GET
- Path: /ws/player/{roomId}/{playerName}
- Notes: playerName should be URL-encoded (spaces -> %20)
- Response 200 (application/json):
  {"name":"Alice","position":"BTN"}
- Response 404 if player not found

WebSocket

Endpoint
- ws://localhost:8080/ws/{roomId}

Client -> Server messages (JSON)

1) Join room
{"type":"JOIN","name":"Alice"}

2) Start game (requires at least 2 players)
{"type":"START","start":true}

3) Player action
{"type":"ACTION","action":"CALL"}
{"type":"ACTION","action":"FOLD"}
{"type":"ACTION","action":"RAISE","amount":100}

4) Chat
{"type":"CHAT","message":"hello"}

5) Advance round (dealer only)
{"type":"NEXT_ROUND"}

6) Remove inactive players (dealer only)
{"type":"KICK_INACTIVE","inactiveSeconds":120}

7) Transfer chips to another player
{"type":"TRANSFER","target":"Bob","amount":50}

Server -> Client messages (JSON)

1) Player joined
{"type":"PLAYER_JOINED","player":"Alice","position":"BTN"}

2) Chat history
{"type":"CHAT_HISTORY","messages":[{"sender":"Alice","message":"hi"}]}

3) Chat message
{"type":"CHAT","sender":"Alice","message":"hi"}

4) Game started
{"type":"GAME_STARTED"}

5) Private hand (per player)
{"type":"PRIVATE_HAND","cards":[{"suit":"espada","rank":"A"}]}

6) Turn
{"type":"TURN","player":"Alice"}

7) Player action broadcast
{"type":"PLAYER_ACTION","player":"Alice","action":"CALL","amount":10,"pot":20}

8) Round advanced
{"type":"ROUND_ADVANCED","round":2,"community":[{"suit":"copa","rank":"K"}]}

9) Hand finished
{"type":"HAND_FINISHED","winner":"Alice","pot":150}

10) Positions updated
{"type":"POSITIONS_UPDATED"}

11) Players removed
{"type":"PLAYERS_REMOVED","players":["Bob","Carol"]}

12) Chips transferred
{"type":"CHIPS_TRANSFERRED","from":"Alice","to":"Bob","amount":50,"fromChips":950,"toChips":1050}

Notes
- Positions are auto-assigned by join order.
- Dealer (BTN) is the only one allowed to advance rounds or remove inactive players.
- Round progression: 2 = flop (3 cards), 3 = turn (1 card), 4 = river (1 card), then hand ends.

erros:
        - fold não funciona
        - fichas do jogador não são alteradas
        - jogo não começa pelo small blind
        - call em caso de ter tido raise não funciona 
        - não tem opção de check