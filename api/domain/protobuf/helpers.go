package protobuf // or whatever package you are using these in

import (
	"time"
)

// Helper to get current time (boilerplate reduction)
func now() int64 {
	return time.Now().UnixMilli()
}

// --- Game Flow & State ---

func MakePacketGameStarted() *ServerPacket {
	return &ServerPacket{
		Payload: &ServerPacket_GameStarted_{
			GameStarted: &ServerPacket_GameStarted{},
		},
		ServerTimestamp: now(),
	}
}

func MakePacketRoundUpdate(roundNumber int64) *ServerPacket {
	return &ServerPacket{
		Payload: &ServerPacket_RoundUpdate_{
			RoundUpdate: &ServerPacket_RoundUpdate{
				RoundNumber: roundNumber,
			},
		},
		ServerTimestamp: now(),
	}
}

func MakePacketInitialRoomSnapshot(players []*ServerPacket_InitialRoomSnapshot_PlayerState, history [][]byte, currentDrawer string, round int32, roomId string, currentPhase int32, nextTick int64) *ServerPacket {
	return &ServerPacket{
		Payload: &ServerPacket_InitialRoomSnapshot_{
			InitialRoomSnapshot: &ServerPacket_InitialRoomSnapshot{
				RoomId:         roomId,
				CurrentPhase:   currentPhase,
				NextTick:       nextTick,
				PlayersStates:  players,
				DrawingHistory: history,
				CurrentDrawer:  currentDrawer,
				CurrentRound:   round,
			},
		},
		ServerTimestamp: now(),
	}
}

func MakePacketLeaderBoard() *ServerPacket {
	return &ServerPacket{
		Payload: &ServerPacket_Leaderboard{ // Watch out for casing here, might be LeaderBoard_ depending on protoc version
			Leaderboard: &ServerPacket_LeaderBoard{},
		},
		ServerTimestamp: now(),
	}
}

// --- Player Actions & Status ---

func MakePacketPlayerJoined(username string) *ServerPacket {
	return &ServerPacket{
		Payload: &ServerPacket_PlayerJoined_{
			PlayerJoined: &ServerPacket_PlayerJoined{
				Username: username,
			},
		},
		ServerTimestamp: now(),
	}
}

func MakePacketPlayerLeft(username string) *ServerPacket {
	return &ServerPacket{
		Payload: &ServerPacket_PlayerLeft_{
			PlayerLeft: &ServerPacket_PlayerLeft{
				Username: username,
			},
		},
		ServerTimestamp: now(),
	}
}

func MakePacketPlayerIsChoosingWord(username string) *ServerPacket {
	return &ServerPacket{
		Payload: &ServerPacket_PlayerIsChoosingWord_{
			PlayerIsChoosingWord: &ServerPacket_PlayerIsChoosingWord{
				Username: username,
			},
		},
		ServerTimestamp: now(),
	}
}

func MakePacketPlayerIsDrawing(username string) *ServerPacket {
	return &ServerPacket{
		Payload: &ServerPacket_PlayerIsDrawing_{
			PlayerIsDrawing: &ServerPacket_PlayerIsDrawing{
				Username: username,
			},
		},
		ServerTimestamp: now(),
	}
}

func MakePacketPlayerGuessedTheWord(username string) *ServerPacket {
	return &ServerPacket{
		Payload: &ServerPacket_PlayerGuessedTheWord_{
			PlayerGuessedTheWord: &ServerPacket_PlayerGuessedTheWord{
				Username: username,
			},
		},
		ServerTimestamp: now(),
	}
}

func MakePacketPlayerMessage(from, msg string) *ServerPacket {
	return &ServerPacket{
		Payload: &ServerPacket_PlayerMessage_{
			PlayerMessage: &ServerPacket_PlayerMessage{
				From:    from,
				Message: msg,
			},
		},
		ServerTimestamp: now(),
	}
}

// --- Gameplay Mechanics ---

func MakePacketDrawingData(data []byte) *ServerPacket {
	return &ServerPacket{
		Payload: &ServerPacket_DrawingData{
			DrawingData: &DrawingData{
				Data: data,
			},
		},
	}
}

func MakePacketPleaseChooseAWord(words []string) *ServerPacket {
	return &ServerPacket{
		Payload: &ServerPacket_PleaseChooseAWord_{
			PleaseChooseAWord: &ServerPacket_PleaseChooseAWord{
				Words: words,
			},
		},
		ServerTimestamp: now(),
	}
}

func MakePacketYourTurnToDraw(word string) *ServerPacket {
	return &ServerPacket{
		Payload: &ServerPacket_YourTurnToDraw_{
			YourTurnToDraw: &ServerPacket_YourTurnToDraw{
				Word: word,
			},
		},
		ServerTimestamp: now(),
	}
}

func MakePacketTurnSummary(wordReveal string, deltas []*ServerPacket_TurnSummary_ScoreDeltas) *ServerPacket {
	return &ServerPacket{
		Payload: &ServerPacket_TurnSummary_{
			TurnSummary: &ServerPacket_TurnSummary{
				WordReveal: wordReveal,
				Deltas:     deltas,
			},
		},
		ServerTimestamp: now(),
	}
}
