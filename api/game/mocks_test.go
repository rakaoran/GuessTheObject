package game

import (
	"api/domain"
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

type MockWebsocketConnection struct {
	mock.Mock
}

func (mt *MockWebsocketConnection) Close() {
	mt.Called()
}
func (mt *MockWebsocketConnection) Ping() error {
	args := mt.Called()
	return args.Error(0)
}
func (mt *MockWebsocketConnection) Read() ([]byte, error) {
	args := mt.Called()
	return args.Get(0).([]byte), args.Error(1)
}

func (mt *MockWebsocketConnection) Write(data []byte) error {
	args := mt.Called(data)
	return args.Error(0)
}

type MockRandomWordsGenerator struct {
	mock.Mock
}

func (m *MockRandomWordsGenerator) Generate(count int) []string {
	args := m.Called(count)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]string)
}

type MockUniqueIdGenerator struct {
	mock.Mock
}

func (m *MockUniqueIdGenerator) Generate() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockUniqueIdGenerator) Dispose(word string) {
	m.Called(word)
}

type MockPeriodicTickerChannelCreator struct {
	mock.Mock
}

func (m *MockPeriodicTickerChannelCreator) Create(duration time.Duration) chan time.Time {
	args := m.Called(duration)
	return args.Get(0).(chan time.Time)
}

// MockUserGetter
type MockUserGetter struct {
	mock.Mock
}

func (m *MockUserGetter) GetUserById(ctx context.Context, id string) (domain.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.User), args.Error(1)
}

type MockPlayer struct {
	mock.Mock
}

func (m *MockPlayer) Send(data []byte) error {
	args := m.Called(data)
	return args.Error(0)
}

func (m *MockPlayer) Ping() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockPlayer) SetRoom(r Room) {
	m.Called(r)
}

func (m *MockPlayer) Cancel() {
	m.Called()
}

func (m *MockPlayer) CancelAndRelease() {
	m.Called()
}
func (m *MockPlayer) Username() string {
	args := m.Called()
	return args.String(0)
}

type MockRoom struct {
	mock.Mock
}

func (m *MockRoom) Description() roomDescription {
	args := m.Called()
	return args.Get(0).(roomDescription)
}

func (m *MockRoom) SetParentLobby(l Lobby) {
	m.Called(l)
}

func (m *MockRoom) CloseAndRelease() {
	m.Called()
}

func (m *MockRoom) PingPlayers() {
	m.Called()
}

func (m *MockRoom) Send(ctx context.Context, e ClientPacketEnvelope) {
	m.Called(ctx, e)
}

func (m *MockRoom) RemoveMe(ctx context.Context, p Player) {
	m.Called(ctx, p)
}

func (m *MockRoom) RequestJoin(jreq roomJoinRequest) {
	m.Called(jreq)
}

func (m *MockRoom) Tick(now time.Time) {
	m.Called(now)
}

func (m *MockRoom) GameLoop() {
	m.Called()
}

func (m *MockRoom) SetId(id string) {
	m.Called(id)
}

type MockLobby struct {
	mock.Mock
}

func (l *MockLobby) RequestUpdateDescription(desc roomDescription) {
	l.Called(desc)

}

func (m *MockLobby) RequestAddAndRunRoom(ctx context.Context, r Room, host Player) {
	m.Called(ctx, r, host)
}

func (m *MockLobby) RemoveRoom(roomId string) {
	m.Called(roomId)
}

func (m *MockLobby) ForwardPlayerJoinRequestToRoom(
	ctx context.Context,
	jreq roomJoinRequest,
) {
	m.Called(ctx, jreq)
}
