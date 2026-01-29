package game

import (
	"api/domain"
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

// --- WebsocketConnection ---

type MockWebsocketConnection struct {
	mock.Mock
}

func (m *MockWebsocketConnection) Close() {
	m.Called()
}

func (m *MockWebsocketConnection) Write(data []byte) error {
	args := m.Called(data)
	return args.Error(0)
}

func (m *MockWebsocketConnection) Read() ([]byte, error) {
	args := m.Called()
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockWebsocketConnection) Ping() error {
	args := m.Called()
	return args.Error(0)
}

// --- RandomWordsGenerator ---

type MockRandomWordsGenerator struct {
	mock.Mock
}

func (m *MockRandomWordsGenerator) Generate(count int) []string {
	args := m.Called(count)
	return args.Get(0).([]string)
}

// --- UniqueIdGenerator ---

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

// --- PeriodicTickerChannelCreator ---

type MockPeriodicTickerChannelCreator struct {
	mock.Mock
}

func (m *MockPeriodicTickerChannelCreator) Create(duration time.Duration) <-chan time.Time {
	args := m.Called(duration)
	return args.Get(0).(chan time.Time)
}

// --- UserGetter ---

type MockUserGetter struct {
	mock.Mock
}

func (m *MockUserGetter) GetUserById(ctx context.Context, id string) (domain.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.User), args.Error(1)
}

// --- Player ---

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

func (m *MockPlayer) CancelAndRelease() {
	m.Called()
}

func (m *MockPlayer) Username() string {
	args := m.Called()
	return args.String(0)
}

// --- Room ---

type MockRoom struct {
	mock.Mock
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

func (m *MockRoom) CloseAndRelease() {
	m.Called()
}

func (m *MockRoom) Description() roomDescription {
	args := m.Called()
	return args.Get(0).(roomDescription)
}

func (m *MockRoom) SetParentLobby(l Lobby) {
	m.Called(l)
}

func (m *MockRoom) SetId(id string) {
	m.Called(id)
}

// --- Lobby ---

type MockLobby struct {
	mock.Mock
}

func (m *MockLobby) RequestAddAndRunRoom(ctx context.Context, r Room) {
	m.Called(ctx, r)
}

func (m *MockLobby) ForwardPlayerJoinRequestToRoom(ctx context.Context, jreq roomJoinRequest) {
	m.Called(ctx, jreq)
}

func (m *MockLobby) RequestUpdateDescription(desc roomDescription) {
	m.Called(desc)
}

func (m *MockLobby) RemoveRoom(roomId string) {
	m.Called(roomId)
}
