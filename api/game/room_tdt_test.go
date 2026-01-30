package game

import (
	"api/domain/protobuf"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/proto"
)

func (st dataSendTask) String() string {
	toName := "<nil>"
	if st.to != nil {
		toName = st.to.Username()
	}
	serverPacket := &protobuf.ServerPacket{}
	if err := proto.Unmarshal(st.data, serverPacket); err != nil {
		return fmt.Sprintf("dataSendTask{to: %s, data: <invalid proto: %v>}", toName, st.data)
	}
	if p, ok := serverPacket.Payload.(*protobuf.ServerPacket_InitialRoomSnapshot_); ok {
		p.InitialRoomSnapshot.NextTick = 0
	}
	return fmt.Sprintf("dataSendTask{to: %s, payload: %+v}", toName, serverPacket.Payload)
}

func MakeDataSendTasks(args ...any) []dataSendTask {
	if len(args)%2 != 0 {
		panic("must provide arguments in pairs!")
	}
	res := make([]dataSendTask, 0, len(args)/2)
	dst := dataSendTask{}

	for i := 0; i < len(args); i += 2 {
		to, ok1 := args[i].(Player)
		serverPacket, ok2 := args[i+1].(*protobuf.ServerPacket)

		if !ok1 || !ok2 {
			panic(fmt.Sprintf("Bad types at index %d, expected (Player, *ServerPacket)", i))
		}

		m, _ := proto.Marshal(serverPacket)

		dst.to = to
		dst.data = m

		res = append(res, dst)
		dst = dataSendTask{}
	}
	return res
}

func AssertEqualDataSendTasks(t *testing.T, expected []dataSendTask, actual []dataSendTask) {
	t.Helper()
	expectedStr := []string{}
	actualStr := []string{}

	for _, d := range expected {
		expectedStr = append(expectedStr, d.String())
	}
	for _, d := range actual {
		actualStr = append(actualStr, d.String())
	}

	assert.ElementsMatch(t, expectedStr, actualStr)
}

func TestGame_GameScenario_1(t *testing.T) {
	t.Parallel()
	naruto := &MockPlayer{}
	naruto.On("Username").Return("naruto")
	sasuke := &MockPlayer{}
	sasuke.On("Username").Return("sasuke")
	sakura := &MockPlayer{}
	itachi := &MockPlayer{}
	itachi.On("Username").Return("itachi")
	jiraiya := &MockPlayer{}
	jiraiya.On("Username").Return("jiraiya")
	itachi2 := &MockPlayer{}
	sasuke.On("SetRoom", mock.Anything).Return().Once()
	naruto.On("SetRoom", mock.Anything).Return().Once()
	itachi.On("SetRoom", mock.Anything).Return().Once()
	jiraiya.On("SetRoom", mock.Anything).Return().Once()

	l := &MockLobby{}
	wordGen := &MockRandomWordsGenerator{}
	r := NewRoom(naruto, false, 4, 2, 3, time.Second*10, time.Second*80, wordGen)
	r.SetId("roomid")
	r.SetId("rid")
	r.SetParentLobby(l)

	now := r.nextTick.Add(-24 * time.Hour)

	testCases := []struct {
		desc                   string
		action                 func()
		setupLobbyExpectations func()
		expectedDataSendTasks  []dataSendTask
		expectedPingSendTasks  []pingSendTask
	}{
		{
			desc: "Sasuke joins",
			action: func() {
				r.handleJoinRequest(roomJoinRequest{player: sasuke, errChan: make(chan error)})
			},
			setupLobbyExpectations: func() {
				l.On("RequestUpdateDescription", roomDescription{
					id: r.id, private: false, playersCount: 2, maxPlayers: r.maxPlayers, started: false,
				}).Return().Once()
			},
			expectedDataSendTasks: MakeDataSendTasks(
				naruto, protobuf.MakePacketPlayerJoined("sasuke"),
				sasuke, protobuf.MakePacketInitialRoomSnapshot([]*protobuf.ServerPacket_InitialRoomSnapshot_PlayerState{{Username: "naruto"}}, [][]byte{}, "", 0, "rid", int32(PHASE_PENDING), r.nextTick.UnixMilli()),
			),
		},
		{
			desc: "itachi joins",
			action: func() {
				r.handleJoinRequest(roomJoinRequest{player: itachi, errChan: make(chan error)})
			},
			setupLobbyExpectations: func() {
				l.On("RequestUpdateDescription", roomDescription{
					id: r.id, private: false, playersCount: 3, maxPlayers: r.maxPlayers, started: false,
				}).Return().Once()
			},
			expectedDataSendTasks: MakeDataSendTasks(
				naruto, protobuf.MakePacketPlayerJoined("itachi"),
				sasuke, protobuf.MakePacketPlayerJoined("itachi"),
				itachi, protobuf.MakePacketInitialRoomSnapshot([]*protobuf.ServerPacket_InitialRoomSnapshot_PlayerState{{Username: "naruto"}, {Username: "sasuke"}}, [][]byte{}, "", 0, "rid", int32(PHASE_PENDING), r.nextTick.UnixMilli()),
			),
		},
		{
			desc: "jiraiya joins",
			action: func() {
				req := roomJoinRequest{player: jiraiya, errChan: make(chan error)}
				r.handleJoinRequest(req)
			},
			setupLobbyExpectations: func() {
				l.On("RequestUpdateDescription", roomDescription{
					id: r.id, private: false, playersCount: 4, maxPlayers: r.maxPlayers, started: false,
				}).Return().Once()
			},
			expectedDataSendTasks: MakeDataSendTasks(
				naruto, protobuf.MakePacketPlayerJoined("jiraiya"),
				sasuke, protobuf.MakePacketPlayerJoined("jiraiya"),
				itachi, protobuf.MakePacketPlayerJoined("jiraiya"),
				jiraiya, protobuf.MakePacketInitialRoomSnapshot([]*protobuf.ServerPacket_InitialRoomSnapshot_PlayerState{{Username: "naruto"}, {Username: "sasuke"}, {Username: "itachi"}}, [][]byte{}, "", 0, "rid", int32(PHASE_PENDING), r.nextTick.UnixMilli()),
			),
		},
		{
			desc: "sakura can't join (room is full)",
			action: func() {
				r.handleJoinRequest(roomJoinRequest{player: sakura, errChan: make(chan error, 1)})
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks:  nil,
		},
		{
			desc: "itachi tries to start game but he's not the host",
			action: func() {
				r.handleStartGameEnvelope("itachi")
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks:  nil,
		},
		{
			desc: "naruto (the host) starts the game",
			action: func() {
				r.handleStartGameEnvelope("naruto")
			},
			setupLobbyExpectations: func() {
				l.On("RequestUpdateDescription", roomDescription{
					id: r.id, private: false, playersCount: 4, maxPlayers: r.maxPlayers, started: true,
				}).Return().Once()
				wordGen.On("Generate", r.wordsCount).Return([]string{"ramen", "kunai", "sharingan"}).Once()
			},
			expectedDataSendTasks: MakeDataSendTasks(
				naruto, protobuf.MakePacketGameStarted(),
				sasuke, protobuf.MakePacketGameStarted(),
				itachi, protobuf.MakePacketGameStarted(),
				jiraiya, protobuf.MakePacketGameStarted(),
				jiraiya, protobuf.MakePacketPleaseChooseAWord([]string{"ramen", "kunai", "sharingan"}),
				naruto, protobuf.MakePacketPlayerIsChoosingWord("jiraiya"),
				sasuke, protobuf.MakePacketPlayerIsChoosingWord("jiraiya"),
				itachi, protobuf.MakePacketPlayerIsChoosingWord("jiraiya"),
			),
		},
		{
			desc: "sasuke leaves",
			action: func() {
				sasuke.On("CancelAndRelease").Return().Once()
				r.handleRemovePlayer(sasuke)
			},
			setupLobbyExpectations: func() {
				l.On("RequestUpdateDescription", roomDescription{
					id: r.id, private: false, playersCount: 3, maxPlayers: r.maxPlayers, started: true,
				}).Return().Once()
			},
			expectedDataSendTasks: MakeDataSendTasks(
				jiraiya, protobuf.MakePacketPlayerLeft("sasuke"),
				itachi, protobuf.MakePacketPlayerLeft("sasuke"),
				naruto, protobuf.MakePacketPlayerLeft("sasuke"),
			),
		},
		{
			desc: "sasuke rejoins after leaving",
			action: func() {
				r.handleJoinRequest(roomJoinRequest{player: sasuke, errChan: make(chan error)})
			},
			setupLobbyExpectations: func() {
				sasuke.On("SetRoom", mock.Anything).Return().Once()
				l.On("RequestUpdateDescription", roomDescription{
					id: r.id, private: false, playersCount: 4, maxPlayers: r.maxPlayers, started: true,
				}).Return().Once()
			},
			expectedDataSendTasks: MakeDataSendTasks(
				naruto, protobuf.MakePacketPlayerJoined("sasuke"),
				jiraiya, protobuf.MakePacketPlayerJoined("sasuke"),
				itachi, protobuf.MakePacketPlayerJoined("sasuke"),
				sasuke, protobuf.MakePacketInitialRoomSnapshot([]*protobuf.ServerPacket_InitialRoomSnapshot_PlayerState{{Username: "naruto"}, {Username: "itachi"}, {Username: "jiraiya"}}, [][]byte{}, "jiraiya", 1, "rid", int32(PHASE_CHOOSING_WORD), r.nextTick.UnixMilli()),
			),
		},
		{
			desc: "sakura still can't join after sasuke rejoined (room is full)",
			action: func() {
				r.handleJoinRequest(roomJoinRequest{player: sakura, errChan: make(chan error, 1)})
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks:  nil,
		},
		{
			desc: "sasuke tries to choose word but he's not the drawer",
			action: func() {
				r.handleWordChoiceEnvelope(&protobuf.ClientPacket_WordChoice{Choice: 1}, "sasuke")
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks:  nil,
		},
		{
			desc: "jiraiya tries to choose an invalid word index (out of bounds)",
			action: func() {
				r.handleWordChoiceEnvelope(&protobuf.ClientPacket_WordChoice{Choice: 99}, "jiraiya")
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks:  nil,
		},
		{
			desc: "jiraiya tries to choose negative index",
			action: func() {
				r.handleWordChoiceEnvelope(&protobuf.ClientPacket_WordChoice{Choice: -1}, "jiraiya")
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks:  nil,
		},
		{
			desc: "jiraiya chooses the word 'kunai' (index 1)",
			action: func() {
				r.handleWordChoiceEnvelope(&protobuf.ClientPacket_WordChoice{Choice: 1}, "jiraiya")
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks: MakeDataSendTasks(
				naruto, protobuf.MakePacketPlayerIsDrawing("jiraiya"),
				sasuke, protobuf.MakePacketPlayerIsDrawing("jiraiya"),
				itachi, protobuf.MakePacketPlayerIsDrawing("jiraiya"),
				jiraiya, protobuf.MakePacketYourTurnToDraw("kunai"),
			),
		},
		{
			desc: "naruto tries to draw but he's not the drawer",
			action: func() {
				drawData := &protobuf.DrawingData{Data: []byte{1, 2, 3}}
				r.handleDrawingDataEnvelope(drawData, "naruto")
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks:  nil,
		},
		{
			desc: "jiraiya draws some data",
			action: func() {
				drawData := &protobuf.DrawingData{Data: []byte{1, 2, 3}}
				r.handleDrawingDataEnvelope(drawData, "jiraiya")
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks: MakeDataSendTasks(
				naruto, protobuf.MakePacketDrawingData([]byte{1, 2, 3}),
				sasuke, protobuf.MakePacketDrawingData([]byte{1, 2, 3}),
				itachi, protobuf.MakePacketDrawingData([]byte{1, 2, 3}),
				jiraiya, protobuf.MakePacketDrawingData([]byte{1, 2, 3}),
			),
		},
		{
			desc: "jiraiya draws more data",
			action: func() {
				drawData := &protobuf.DrawingData{Data: []byte{4, 5, 6}}
				r.handleDrawingDataEnvelope(drawData, "jiraiya")
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks: MakeDataSendTasks(
				naruto, protobuf.MakePacketDrawingData([]byte{4, 5, 6}),
				sasuke, protobuf.MakePacketDrawingData([]byte{4, 5, 6}),
				itachi, protobuf.MakePacketDrawingData([]byte{4, 5, 6}),
				jiraiya, protobuf.MakePacketDrawingData([]byte{4, 5, 6}),
			),
		},
		{
			desc: "naruto guesses wrong",
			action: func() {
				r.handlePlayerMessageEnvelope(&protobuf.ClientPacket_PlayerMessage{Message: "shuriken"}, "naruto")
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks: MakeDataSendTasks(
				sasuke, protobuf.MakePacketPlayerMessage("naruto", "shuriken"),
				itachi, protobuf.MakePacketPlayerMessage("naruto", "shuriken"),
				jiraiya, protobuf.MakePacketPlayerMessage("naruto", "shuriken"),
			),
		},
		{
			desc: "itachi guesses correctly",
			action: func() {
				r.handlePlayerMessageEnvelope(&protobuf.ClientPacket_PlayerMessage{Message: "kunai"}, "itachi")
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks: MakeDataSendTasks(
				naruto, protobuf.MakePacketPlayerGuessedTheWord("itachi"),
				sasuke, protobuf.MakePacketPlayerGuessedTheWord("itachi"),
				itachi, protobuf.MakePacketPlayerGuessedTheWord("itachi"),
				jiraiya, protobuf.MakePacketPlayerGuessedTheWord("itachi"),
			),
		},
		{
			desc: "itachi sends a message after guessing (only visible to drawer and other guessers)",
			action: func() {
				r.handlePlayerMessageEnvelope(&protobuf.ClientPacket_PlayerMessage{Message: "ez clap"}, "itachi")
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks: MakeDataSendTasks(
				jiraiya, protobuf.MakePacketPlayerMessage("itachi", "ez clap"),
			),
		},
		{
			desc: "sasuke also guesses correctly",
			action: func() {
				r.handlePlayerMessageEnvelope(&protobuf.ClientPacket_PlayerMessage{Message: "kunai"}, "sasuke")
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks: MakeDataSendTasks(
				naruto, protobuf.MakePacketPlayerGuessedTheWord("sasuke"),
				sasuke, protobuf.MakePacketPlayerGuessedTheWord("sasuke"),
				itachi, protobuf.MakePacketPlayerGuessedTheWord("sasuke"),
				jiraiya, protobuf.MakePacketPlayerGuessedTheWord("sasuke"),
			),
		},
		{
			desc: "sasuke messages after guessing (visible to drawer and other guessers)",
			action: func() {
				r.handlePlayerMessageEnvelope(&protobuf.ClientPacket_PlayerMessage{Message: "gg"}, "sasuke")
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks: MakeDataSendTasks(
				itachi, protobuf.MakePacketPlayerMessage("sasuke", "gg"),
				jiraiya, protobuf.MakePacketPlayerMessage("sasuke", "gg"),
			),
		},
		{
			desc: "itachi tries to guess again (already guessed, should send message to guessers only)",
			action: func() {
				r.handlePlayerMessageEnvelope(&protobuf.ClientPacket_PlayerMessage{Message: "kunai"}, "itachi")
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks: MakeDataSendTasks(
				sasuke, protobuf.MakePacketPlayerMessage("itachi", "kunai"),
				jiraiya, protobuf.MakePacketPlayerMessage("itachi", "kunai"),
			),
		},
		{
			desc: "naruto still chatting before guessing (visible to everyone)",
			action: func() {
				r.handlePlayerMessageEnvelope(&protobuf.ClientPacket_PlayerMessage{Message: "hmm what is it"}, "naruto")
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks: MakeDataSendTasks(
				sasuke, protobuf.MakePacketPlayerMessage("naruto", "hmm what is it"),
				itachi, protobuf.MakePacketPlayerMessage("naruto", "hmm what is it"),
				jiraiya, protobuf.MakePacketPlayerMessage("naruto", "hmm what is it"),
			),
		},
		{
			desc: "naruto finally guesses correctly (all guessers found, should transition to turn summary)",
			action: func() {
				r.handlePlayerMessageEnvelope(&protobuf.ClientPacket_PlayerMessage{Message: "kunai"}, "naruto")
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks: MakeDataSendTasks(
				naruto, protobuf.MakePacketPlayerGuessedTheWord("naruto"),
				sasuke, protobuf.MakePacketPlayerGuessedTheWord("naruto"),
				itachi, protobuf.MakePacketPlayerGuessedTheWord("naruto"),
				jiraiya, protobuf.MakePacketPlayerGuessedTheWord("naruto"),
				// Turn summary should follow but it's triggered internally
				naruto, protobuf.MakePacketTurnSummary("kunai", []*protobuf.ServerPacket_TurnSummary_ScoreDeltas{
					{Username: "naruto", ScoreDelta: 100},
					{Username: "itachi", ScoreDelta: 300},
					{Username: "jiraiya", ScoreDelta: 0},
					{Username: "sasuke", ScoreDelta: 200},
				}),
				sasuke, protobuf.MakePacketTurnSummary("kunai", []*protobuf.ServerPacket_TurnSummary_ScoreDeltas{
					{Username: "naruto", ScoreDelta: 100},
					{Username: "itachi", ScoreDelta: 300},
					{Username: "jiraiya", ScoreDelta: 0},
					{Username: "sasuke", ScoreDelta: 200},
				}),
				itachi, protobuf.MakePacketTurnSummary("kunai", []*protobuf.ServerPacket_TurnSummary_ScoreDeltas{
					{Username: "naruto", ScoreDelta: 100},
					{Username: "itachi", ScoreDelta: 300},
					{Username: "jiraiya", ScoreDelta: 0},
					{Username: "sasuke", ScoreDelta: 200},
				}),
				jiraiya, protobuf.MakePacketTurnSummary("kunai", []*protobuf.ServerPacket_TurnSummary_ScoreDeltas{
					{Username: "naruto", ScoreDelta: 100},
					{Username: "itachi", ScoreDelta: 300},
					{Username: "jiraiya", ScoreDelta: 0},
					{Username: "sasuke", ScoreDelta: 200},
				}),
			),
		},
		{
			desc: "tick to transition from turn summary to choosing word (next drawer is itachi)",
			action: func() {
				now = now.Add(6 * time.Second)
				r.handleTick(now)
			},
			setupLobbyExpectations: func() {
				wordGen.On("Generate", r.wordsCount).Return([]string{"rasengan", "scroll", "hokage"}).Once()
			},
			expectedDataSendTasks: MakeDataSendTasks(
				itachi, protobuf.MakePacketPleaseChooseAWord([]string{"rasengan", "scroll", "hokage"}),
				naruto, protobuf.MakePacketPlayerIsChoosingWord("itachi"),
				sasuke, protobuf.MakePacketPlayerIsChoosingWord("itachi"),
				jiraiya, protobuf.MakePacketPlayerIsChoosingWord("itachi"),
			),
		},
		{
			desc: "itachi chooses 'rasengan' (index 0)",
			action: func() {
				r.handleWordChoiceEnvelope(&protobuf.ClientPacket_WordChoice{Choice: 0}, "itachi")
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks: MakeDataSendTasks(
				naruto, protobuf.MakePacketPlayerIsDrawing("itachi"),
				sasuke, protobuf.MakePacketPlayerIsDrawing("itachi"),
				jiraiya, protobuf.MakePacketPlayerIsDrawing("itachi"),
				itachi, protobuf.MakePacketYourTurnToDraw("rasengan"),
			),
		},
		{
			desc: "only sasuke guesses correctly this time",
			action: func() {
				r.handlePlayerMessageEnvelope(&protobuf.ClientPacket_PlayerMessage{Message: "rasengan"}, "sasuke")
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks: MakeDataSendTasks(
				naruto, protobuf.MakePacketPlayerGuessedTheWord("sasuke"),
				sasuke, protobuf.MakePacketPlayerGuessedTheWord("sasuke"),
				itachi, protobuf.MakePacketPlayerGuessedTheWord("sasuke"),
				jiraiya, protobuf.MakePacketPlayerGuessedTheWord("sasuke"),
			),
		},
		{
			desc: "tick forward but before drawing ends (no transition yet)",
			action: func() {
				now = now.Add(50 * time.Second)
				r.handleTick(now)
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks:  nil,
		},
		{
			desc: "tick to end drawing phase (81 seconds passed, transition to turn summary)",
			action: func() {
				now = now.Add(31 * time.Second)
				r.handleTick(now)
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks: MakeDataSendTasks(
				naruto, protobuf.MakePacketTurnSummary("rasengan", []*protobuf.ServerPacket_TurnSummary_ScoreDeltas{
					{Username: "naruto", ScoreDelta: 0},
					{Username: "itachi", ScoreDelta: 0},
					{Username: "jiraiya", ScoreDelta: 0},
					{Username: "sasuke", ScoreDelta: 300},
				}),
				sasuke, protobuf.MakePacketTurnSummary("rasengan", []*protobuf.ServerPacket_TurnSummary_ScoreDeltas{
					{Username: "naruto", ScoreDelta: 0},
					{Username: "itachi", ScoreDelta: 0},
					{Username: "jiraiya", ScoreDelta: 0},
					{Username: "sasuke", ScoreDelta: 300},
				}),
				itachi, protobuf.MakePacketTurnSummary("rasengan", []*protobuf.ServerPacket_TurnSummary_ScoreDeltas{
					{Username: "naruto", ScoreDelta: 0},
					{Username: "itachi", ScoreDelta: 0},
					{Username: "jiraiya", ScoreDelta: 0},
					{Username: "sasuke", ScoreDelta: 300},
				}),
				jiraiya, protobuf.MakePacketTurnSummary("rasengan", []*protobuf.ServerPacket_TurnSummary_ScoreDeltas{
					{Username: "naruto", ScoreDelta: 0},
					{Username: "itachi", ScoreDelta: 0},
					{Username: "jiraiya", ScoreDelta: 0},
					{Username: "sasuke", ScoreDelta: 300},
				}),
			),
		},
		{
			desc: "tick to transition to next turn (naruto's turn)",
			action: func() {
				now = now.Add(6 * time.Second)
				r.handleTick(now)
			},
			setupLobbyExpectations: func() {
				wordGen.On("Generate", r.wordsCount).Return([]string{"byakugan", "chidori", "amaterasu"}).Once()
			},
			expectedDataSendTasks: MakeDataSendTasks(
				naruto, protobuf.MakePacketPleaseChooseAWord([]string{"byakugan", "chidori", "amaterasu"}),
				sasuke, protobuf.MakePacketPlayerIsChoosingWord("naruto"),
				itachi, protobuf.MakePacketPlayerIsChoosingWord("naruto"),
				jiraiya, protobuf.MakePacketPlayerIsChoosingWord("naruto"),
			),
		},
		{
			desc: "naruto disconnects while choosing word (should transition immediately to next round)",
			action: func() {
				r.handleRemovePlayer(naruto)
			},
			setupLobbyExpectations: func() {
				naruto.On("CancelAndRelease").Return().Once()
				// jiraiya.On("CancelAndRelease").Return().Once()
				l.On("RequestUpdateDescription", roomDescription{
					id: r.id, private: false, playersCount: 3, maxPlayers: r.maxPlayers, started: true,
				}).Return().Once()
				wordGen.On("Generate", r.wordsCount).Return([]string{"sakura", "kakashi", "zabuza"}).Once()
			},
			expectedDataSendTasks: MakeDataSendTasks(
				sasuke, protobuf.MakePacketPlayerLeft("naruto"),
				itachi, protobuf.MakePacketPlayerLeft("naruto"),
				jiraiya, protobuf.MakePacketPlayerLeft("naruto"),

				sasuke, protobuf.MakePacketRoundUpdate(2),
				itachi, protobuf.MakePacketRoundUpdate(2),
				jiraiya, protobuf.MakePacketRoundUpdate(2),

				sasuke, protobuf.MakePacketPleaseChooseAWord([]string{"sakura", "kakashi", "zabuza"}),
				itachi, protobuf.MakePacketPlayerIsChoosingWord("sasuke"),
				jiraiya, protobuf.MakePacketPlayerIsChoosingWord("sasuke"),
			),
		},
		{
			desc: "sasuke chooses word and draws",
			action: func() {
				r.handleWordChoiceEnvelope(&protobuf.ClientPacket_WordChoice{Choice: 1}, "sasuke")
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks: MakeDataSendTasks(
				itachi, protobuf.MakePacketPlayerIsDrawing("sasuke"),
				jiraiya, protobuf.MakePacketPlayerIsDrawing("sasuke"),
				sasuke, protobuf.MakePacketYourTurnToDraw("kakashi"),
			),
		},
		{
			desc: "itachi guesses correctly first",
			action: func() {
				r.handlePlayerMessageEnvelope(&protobuf.ClientPacket_PlayerMessage{Message: "kakashi"}, "itachi")
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks: MakeDataSendTasks(
				sasuke, protobuf.MakePacketPlayerGuessedTheWord("itachi"),
				itachi, protobuf.MakePacketPlayerGuessedTheWord("itachi"),
				jiraiya, protobuf.MakePacketPlayerGuessedTheWord("itachi"),
			),
		},
		{
			desc: "jiraiya guesses too (all guessed, should transition to turn summary)",
			action: func() {
				r.handlePlayerMessageEnvelope(&protobuf.ClientPacket_PlayerMessage{Message: "kakashi"}, "jiraiya")
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks: MakeDataSendTasks(
				sasuke, protobuf.MakePacketPlayerGuessedTheWord("jiraiya"),
				itachi, protobuf.MakePacketPlayerGuessedTheWord("jiraiya"),
				jiraiya, protobuf.MakePacketPlayerGuessedTheWord("jiraiya"),
				sasuke, protobuf.MakePacketTurnSummary("kakashi", []*protobuf.ServerPacket_TurnSummary_ScoreDeltas{
					{Username: "itachi", ScoreDelta: 200},
					{Username: "jiraiya", ScoreDelta: 100},
					{Username: "sasuke", ScoreDelta: 0},
				}),
				itachi, protobuf.MakePacketTurnSummary("kakashi", []*protobuf.ServerPacket_TurnSummary_ScoreDeltas{
					{Username: "itachi", ScoreDelta: 200},
					{Username: "jiraiya", ScoreDelta: 100},
					{Username: "sasuke", ScoreDelta: 0},
				}),
				jiraiya, protobuf.MakePacketTurnSummary("kakashi", []*protobuf.ServerPacket_TurnSummary_ScoreDeltas{
					{Username: "itachi", ScoreDelta: 200},
					{Username: "jiraiya", ScoreDelta: 100},
					{Username: "sasuke", ScoreDelta: 0},
				}),
			),
		},
		{
			desc: "tick to transition to round 2 (drawerIndex wraps to 0, round increments)",
			action: func() {
				now = now.Add(6 * time.Second)
				r.handleTick(now)
			},
			setupLobbyExpectations: func() {
				wordGen.On("Generate", r.wordsCount).Return([]string{"nine-tails", "sage-mode", "shadow-clone"}).Once()
			},
			expectedDataSendTasks: MakeDataSendTasks(
				jiraiya, protobuf.MakePacketPleaseChooseAWord([]string{"nine-tails", "sage-mode", "shadow-clone"}),
				sasuke, protobuf.MakePacketPlayerIsChoosingWord("jiraiya"),
				itachi, protobuf.MakePacketPlayerIsChoosingWord("jiraiya"),
			),
		},
		{
			desc: "jiraiya chooses word for round 2",
			action: func() {
				r.handleWordChoiceEnvelope(&protobuf.ClientPacket_WordChoice{Choice: 2}, "jiraiya")
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks: MakeDataSendTasks(
				sasuke, protobuf.MakePacketPlayerIsDrawing("jiraiya"),
				itachi, protobuf.MakePacketPlayerIsDrawing("jiraiya"),
				jiraiya, protobuf.MakePacketYourTurnToDraw("shadow-clone"),
			),
		},
		{
			desc: "itachi 2 (player with same username) tries to join - original itachi should be removed first",
			action: func() {

				itachi2.On("Username").Return("itachi")
				itachi2.On("SetRoom", mock.Anything).Return().Once()

				// In real implementation, the system should detect duplicate username
				// For now, we manually remove the original naruto and add naruto2
				itachi.On("CancelAndRelease").Return().Once()
				r.handleJoinRequest(roomJoinRequest{player: itachi2, errChan: make(chan error)})
			},
			setupLobbyExpectations: func() {
				// First removal
				l.On("RequestUpdateDescription", roomDescription{
					id: r.id, private: false, playersCount: 2, maxPlayers: r.maxPlayers, started: true,
				}).Return().Once()
				// Then addition
				l.On("RequestUpdateDescription", roomDescription{
					id: r.id, private: false, playersCount: 3, maxPlayers: r.maxPlayers, started: true,
				}).Return().Once()
			},
			expectedDataSendTasks: MakeDataSendTasks(
				jiraiya, protobuf.MakePacketPlayerLeft("itachi"),
				sasuke, protobuf.MakePacketPlayerLeft("itachi"),
				jiraiya, protobuf.MakePacketPlayerJoined("itachi"),
				sasuke, protobuf.MakePacketPlayerJoined("itachi"),
				itachi2, protobuf.MakePacketInitialRoomSnapshot([]*protobuf.ServerPacket_InitialRoomSnapshot_PlayerState{{Username: "jiraiya", Score: 100}, {Username: "sasuke", Score: 500}}, [][]byte{}, "jiraiya", 2, "rid", int32(PHASE_DRAWING), r.nextTick.UnixMilli()),
			),
		},
		{
			desc: "itachi2 guesses correctly first",
			action: func() {
				r.handlePlayerMessageEnvelope(&protobuf.ClientPacket_PlayerMessage{Message: "shadow-clone"}, "itachi")
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks: MakeDataSendTasks(
				sasuke, protobuf.MakePacketPlayerGuessedTheWord("itachi"),
				jiraiya, protobuf.MakePacketPlayerGuessedTheWord("itachi"),
				itachi2, protobuf.MakePacketPlayerGuessedTheWord("itachi"),
			),
		},
		{
			desc: "sasuke guesses correctly",
			action: func() {
				r.handlePlayerMessageEnvelope(&protobuf.ClientPacket_PlayerMessage{Message: "shadow-clone"}, "sasuke")
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks: MakeDataSendTasks(
				sasuke, protobuf.MakePacketPlayerGuessedTheWord("sasuke"),
				jiraiya, protobuf.MakePacketPlayerGuessedTheWord("sasuke"),
				itachi2, protobuf.MakePacketPlayerGuessedTheWord("sasuke"),
				sasuke, protobuf.MakePacketTurnSummary(
					"shadow-clone", []*protobuf.ServerPacket_TurnSummary_ScoreDeltas{
						{Username: "jiraiya", ScoreDelta: 0},
						{Username: "sasuke", ScoreDelta: 100},
						{Username: "itachi", ScoreDelta: 200},
					},
				),
				jiraiya, protobuf.MakePacketTurnSummary(
					"shadow-clone", []*protobuf.ServerPacket_TurnSummary_ScoreDeltas{
						{Username: "jiraiya", ScoreDelta: 0},
						{Username: "sasuke", ScoreDelta: 100},
						{Username: "itachi", ScoreDelta: 200},
					},
				),
				itachi2, protobuf.MakePacketTurnSummary(
					"shadow-clone", []*protobuf.ServerPacket_TurnSummary_ScoreDeltas{
						{Username: "jiraiya", ScoreDelta: 0},
						{Username: "sasuke", ScoreDelta: 100},
						{Username: "itachi", ScoreDelta: 200},
					},
				),
			),
		},
		{
			desc: "leaderboard",
			action: func() {
				r.handleTick(now.Add(6 * time.Second))
			},
			setupLobbyExpectations: func() {
				l.On("RemoveRoom", "rid").Return().Once()
				sasuke.On("CancelAndRelease").Return().Once()
				jiraiya.On("CancelAndRelease").Return().Once()
				itachi2.On("CancelAndRelease").Return().Once()
			},
			expectedDataSendTasks: MakeDataSendTasks(
				sasuke, protobuf.MakePacketLeaderBoard(),
				jiraiya, protobuf.MakePacketLeaderBoard(),
				itachi2, protobuf.MakePacketLeaderBoard(),
			),
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			tC.setupLobbyExpectations()
			tC.action()
			if tC.expectedDataSendTasks != nil {
				AssertEqualDataSendTasks(t, tC.expectedDataSendTasks, r.dataSendTasks)
			}
			r.dataSendTasks = make([]dataSendTask, 0)
			r.pingSendTasks = make([]pingSendTask, 0)
		})
	}

	l.AssertExpectations(t)
	wordGen.AssertExpectations(t)
	naruto.AssertExpectations(t)
	sasuke.AssertExpectations(t)
	itachi.AssertExpectations(t)

	jiraiya.AssertExpectations(t)
}
