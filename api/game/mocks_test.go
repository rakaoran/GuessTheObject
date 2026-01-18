package game_test

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

// MockRandomWordsGenerator
type MockRandomWordsGenerator struct {
	mock.Mock
}

func (m *MockRandomWordsGenerator) Generate(count int) []string {
	args := m.Called(count)
	// Careful here: if you don't return a slice in your test setup (Return(nil)),
	// this type assertion will panic.
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]string)
}

// MockUniqueIdGenerator
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

// MockPeriodicTickerChannelCreator
type MockPeriodicTickerChannelCreator struct {
	mock.Mock
}

func (m *MockPeriodicTickerChannelCreator) Create() <-chan time.Time {
	args := m.Called()
	// You have to cast the generic return to the specific channel type
	return args.Get(0).(<-chan time.Time)
}

// MockUserGetter
type MockUserGetter struct {
	mock.Mock
}

func (m *MockUserGetter) GetUserById(ctx context.Context, id string) (domain.User, error) {
	args := m.Called(ctx, id)
	// Assuming args.Get(0) is actually a domain.User struct.
	// If you return nil in tests for the struct, this will panic because you can't cast nil to a struct value.
	return args.Get(0).(domain.User), args.Error(1)
}
