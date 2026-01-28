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
	sakura.On("Username").Return("sakura")
	itachi := &MockPlayer{}
	itachi.On("Username").Return("itachi")
	jiraiya := &MockPlayer{}
	jiraiya.On("Username").Return("jiraiya")
	sasuke.On("SetRoom", mock.Anything).Return().Once()
	naruto.On("SetRoom", mock.Anything).Return().Once()
	itachi.On("SetRoom", mock.Anything).Return().Once()
	jiraiya.On("SetRoom", mock.Anything).Return().Once()
	// sakura.On("SetRoom", mock.Anything).Return()

	l := &MockLobby{}
	wordGen := &MockRandomWordsGenerator{}
	r := NewRoom(naruto, false, 4, 2, 3, time.Second*10, time.Second*80, wordGen)
	r.SetId("roomid")
	r.SetId("rid")
	r.SetParentLobby(l)

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
				sasuke, protobuf.MakePacketInitialRoomSnapshot([]*protobuf.ServerPacket_InitialRoomSnapshot_PlayerState{{Username: "naruto"}}, [][]byte{}, "", 0, r.id, int32(r.phase), r.nextTick.UnixMilli()),
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
				itachi, protobuf.MakePacketInitialRoomSnapshot([]*protobuf.ServerPacket_InitialRoomSnapshot_PlayerState{{Username: "naruto"}, {Username: "sasuke"}}, [][]byte{}, "", 0, r.id, int32(r.phase), r.nextTick.UnixMilli()),
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
				jiraiya, protobuf.MakePacketInitialRoomSnapshot([]*protobuf.ServerPacket_InitialRoomSnapshot_PlayerState{{Username: "naruto"}, {Username: "sasuke"}, {Username: "itachi"}}, [][]byte{}, "", 0, r.id, int32(r.phase), r.nextTick.UnixMilli()),
			),
		},
		{
			desc: "sakura can't join",
			action: func() {
				r.handleJoinRequest(roomJoinRequest{player: sakura, errChan: make(chan error, 1)})
			},
			setupLobbyExpectations: func() {},
			expectedDataSendTasks:  nil,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			tC.setupLobbyExpectations()
			tC.action()
			AssertEqualDataSendTasks(t, tC.expectedDataSendTasks, r.dataSendTasks)
			r.dataSendTasks = make([]dataSendTask, 0)
			r.pingSendTasks = make([]pingSendTask, 0)
		})
	}

	l.AssertExpectations(t)
	wordGen.AssertExpectations(t)
	naruto.AssertExpectations(t)
	sasuke.AssertExpectations(t)
	// sakura.AssertExpectations(t)
	// itachi.AssertExpectations(t)
	// jiraiya.AssertExpectations(t)
}

// TODO: players with same username join

// Gonna add more test scenarios if I got time lol
