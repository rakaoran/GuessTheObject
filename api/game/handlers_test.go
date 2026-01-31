package game

import (
	"api/domain"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateGameHandler_Validation(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	baseQuery := "maxPlayers=5&roundsCount=3&wordsCount=3&choosingWordDuration=30&drawingDuration=60"

	testCases := []struct {
		name         string
		setupMocks   func(*MockLobby, *MockUserGetter)
		query        string
		userId       string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "missing user id",
			setupMocks:   func(l *MockLobby, u *MockUserGetter) {},
			query:        baseQuery,
			userId:       "",
			expectedCode: http.StatusUnauthorized,
			expectedBody: "unauthenticated",
		},
		{
			name:         "invalid parameter type",
			setupMocks:   func(l *MockLobby, u *MockUserGetter) {},
			query:        "maxPlayers=not-a-number&roundsCount=3",
			userId:       "user-123",
			expectedCode: http.StatusBadRequest,
			expectedBody: "invalid-request-format",
		},
		{
			name:         "maxPlayers too low",
			setupMocks:   func(l *MockLobby, u *MockUserGetter) {},
			query:        "maxPlayers=1&roundsCount=3&wordsCount=3&choosingWordDuration=30&drawingDuration=60",
			userId:       "user-123",
			expectedCode: http.StatusBadRequest,
			expectedBody: "maxPlayers must be at least 2",
		},
		{
			name:         "maxPlayers too high",
			setupMocks:   func(l *MockLobby, u *MockUserGetter) {},
			query:        "maxPlayers=21&roundsCount=3&wordsCount=3&choosingWordDuration=30&drawingDuration=60",
			userId:       "user-123",
			expectedCode: http.StatusBadRequest,
			expectedBody: "maxPlayers cannot exceed 20",
		},
		{
			name:         "roundsCount too low",
			setupMocks:   func(l *MockLobby, u *MockUserGetter) {},
			query:        "maxPlayers=5&roundsCount=0&wordsCount=3&choosingWordDuration=30&drawingDuration=60",
			userId:       "user-123",
			expectedCode: http.StatusBadRequest,
			expectedBody: "roundsCount must be at least 1",
		},
		{
			name:         "roundsCount too high",
			setupMocks:   func(l *MockLobby, u *MockUserGetter) {},
			query:        "maxPlayers=5&roundsCount=11&wordsCount=3&choosingWordDuration=30&drawingDuration=60",
			userId:       "user-123",
			expectedCode: http.StatusBadRequest,
			expectedBody: "roundsCount cannot exceed 10",
		},
		{
			name:         "wordsCount too low",
			setupMocks:   func(l *MockLobby, u *MockUserGetter) {},
			query:        "maxPlayers=5&roundsCount=3&wordsCount=0&choosingWordDuration=30&drawingDuration=60",
			userId:       "user-123",
			expectedCode: http.StatusBadRequest,
			expectedBody: "wordsCount must be at least 1",
		},
		{
			name:         "wordsCount too high",
			setupMocks:   func(l *MockLobby, u *MockUserGetter) {},
			query:        "maxPlayers=5&roundsCount=3&wordsCount=6&choosingWordDuration=30&drawingDuration=60",
			userId:       "user-123",
			expectedCode: http.StatusBadRequest,
			expectedBody: "wordsCount cannot exceed 5",
		},
		{
			name:         "choosingWordDuration too low",
			setupMocks:   func(l *MockLobby, u *MockUserGetter) {},
			query:        "maxPlayers=5&roundsCount=3&wordsCount=3&choosingWordDuration=4&drawingDuration=60",
			userId:       "user-123",
			expectedCode: http.StatusBadRequest,
			expectedBody: "choosingWordDuration must be at least 5 seconds",
		},
		{
			name:         "choosingWordDuration too high",
			setupMocks:   func(l *MockLobby, u *MockUserGetter) {},
			query:        "maxPlayers=5&roundsCount=3&wordsCount=3&choosingWordDuration=121&drawingDuration=60",
			userId:       "user-123",
			expectedCode: http.StatusBadRequest,
			expectedBody: "choosingWordDuration cannot exceed 120 seconds",
		},
		{
			name:         "drawingDuration too low",
			setupMocks:   func(l *MockLobby, u *MockUserGetter) {},
			query:        "maxPlayers=5&roundsCount=3&wordsCount=3&choosingWordDuration=30&drawingDuration=29",
			userId:       "user-123",
			expectedCode: http.StatusBadRequest,
			expectedBody: "drawingDuration must be at least 30 seconds",
		},
		{
			name:         "drawingDuration too high",
			setupMocks:   func(l *MockLobby, u *MockUserGetter) {},
			query:        "maxPlayers=5&roundsCount=3&wordsCount=3&choosingWordDuration=30&drawingDuration=301",
			userId:       "user-123",
			expectedCode: http.StatusBadRequest,
			expectedBody: "drawingDuration cannot exceed 300 seconds",
		},
		{
			name: "user not found",
			setupMocks: func(l *MockLobby, u *MockUserGetter) {
				u.On("GetUserById", mock.Anything, "user-123").Return(domain.User{}, domain.ErrUserNotFound)
			},
			query:        baseQuery,
			userId:       "user-123",
			expectedCode: http.StatusUnauthorized,
			expectedBody: "user-not-found",
		},
		{
			name: "database error",
			setupMocks: func(l *MockLobby, u *MockUserGetter) {
				u.On("GetUserById", mock.Anything, "user-123").Return(domain.User{}, errors.New("db error"))
			},
			query:        baseQuery,
			userId:       "user-123",
			expectedCode: http.StatusInternalServerError,
			expectedBody: "failed-to-get-user",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockLobby := &MockLobby{}
			mockUserGetter := &MockUserGetter{}
			mockWordGen := &MockRandomWordsGenerator{}

			tc.setupMocks(mockLobby, mockUserGetter)

			handler := NewGameHandler(mockLobby, mockUserGetter, mockWordGen)

			router := gin.New()
			router.GET("/create", func(c *gin.Context) {
				if tc.userId != "" {
					c.Set("id", tc.userId)
				}
				handler.CreateGameHandler(c)
			})

			target := "/create?" + tc.query
			req := httptest.NewRequest(http.MethodGet, target, nil)

			res := httptest.NewRecorder()

			router.ServeHTTP(res, req)

			assert.Equal(t, tc.expectedCode, res.Code)
			assert.Contains(t, res.Body.String(), tc.expectedBody)

			mockLobby.AssertExpectations(t)
			mockUserGetter.AssertExpectations(t)
		})
	}
}
func TestJoinGameHandler_Validation(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name         string
		setupMocks   func(*MockLobby, *MockUserGetter)
		userId       string
		roomId       string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "missing user id",
			setupMocks:   func(l *MockLobby, u *MockUserGetter) {},
			userId:       "",
			roomId:       "ROOM1",
			expectedCode: http.StatusUnauthorized,
			expectedBody: "unauthenticated",
		},
		{
			name:         "empty room id - 404",
			setupMocks:   func(l *MockLobby, u *MockUserGetter) {},
			userId:       "user-123",
			roomId:       "",
			expectedCode: http.StatusNotFound, // Route doesn't match
			expectedBody: "404",
		},
		{
			name: "user not found",
			setupMocks: func(l *MockLobby, u *MockUserGetter) {
				u.On("GetUserById", mock.Anything, "user-123").Return(domain.User{}, domain.ErrUserNotFound)
			},
			userId:       "user-123",
			roomId:       "ROOM1",
			expectedCode: http.StatusUnauthorized,
			expectedBody: "user-not-found",
		},
		{
			name: "database error",
			setupMocks: func(l *MockLobby, u *MockUserGetter) {
				u.On("GetUserById", mock.Anything, "user-123").Return(domain.User{}, errors.New("db error"))
			},
			userId:       "user-123",
			roomId:       "ROOM1",
			expectedCode: http.StatusInternalServerError,
			expectedBody: "failed-to-get-user",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockLobby := &MockLobby{}
			mockUserGetter := &MockUserGetter{}
			mockWordGen := &MockRandomWordsGenerator{}

			tc.setupMocks(mockLobby, mockUserGetter)

			handler := NewGameHandler(mockLobby, mockUserGetter, mockWordGen)

			router := gin.New()
			router.GET("/join/:roomid", func(c *gin.Context) {
				if tc.userId != "" {
					c.Set("id", tc.userId)
				}
				handler.JoinGameHandler(c)
			})

			path := "/join/" + tc.roomId
			req := httptest.NewRequest(http.MethodGet, path, nil)
			res := httptest.NewRecorder()

			router.ServeHTTP(res, req)

			assert.Equal(t, tc.expectedCode, res.Code)
			if tc.expectedBody != "" {
				assert.Contains(t, res.Body.String(), tc.expectedBody)
			}

			mockLobby.AssertExpectations(t)
			mockUserGetter.AssertExpectations(t)
		})
	}
}

func TestGorillaWebSocketWrapper(t *testing.T) {
	t.Parallel()

	t.Run("read and write", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			upgrader := websocket.Upgrader{
				CheckOrigin: func(r *http.Request) bool { return true },
			}
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()

			wrapper := NewGorillaWebSocketWrapper(conn)

			data, err := wrapper.Read()
			if err != nil {
				return
			}

			wrapper.Write(data)
		}))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		assert.NoError(t, err)
		defer conn.Close()

		testData := []byte("test message")
		conn.WriteMessage(websocket.BinaryMessage, testData)

		_, msg, err := conn.ReadMessage()
		assert.NoError(t, err)
		assert.Equal(t, testData, msg)
	})

	t.Run("ping", func(t *testing.T) {
		t.Parallel()

		done := make(chan struct{})
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			upgrader := websocket.Upgrader{
				CheckOrigin: func(r *http.Request) bool { return true },
			}
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()

			wrapper := NewGorillaWebSocketWrapper(conn)
			wrapper.Ping()

			<-done
		}))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		assert.NoError(t, err)
		defer conn.Close()

		time.Sleep(50 * time.Millisecond)
		close(done)
	})

	t.Run("close", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			upgrader := websocket.Upgrader{
				CheckOrigin: func(r *http.Request) bool { return true },
			}
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}

			wrapper := NewGorillaWebSocketWrapper(conn)
			time.Sleep(50 * time.Millisecond)
			wrapper.Close()
		}))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		assert.NoError(t, err)
		defer conn.Close()

		conn.SetReadDeadline(time.Now().Add(time.Second))
		_, _, err = conn.ReadMessage()
		assert.Error(t, err)
	})
}

func TestCreateGameHandler_Success(t *testing.T) {
	mockLobby := &MockLobby{}

	mockUserGetter := &MockUserGetter{}
	mockWordGen := &MockRandomWordsGenerator{}

	user := domain.User{Id: "user-123", Username: "HostPlayer"}
	mockUserGetter.On("GetUserById", mock.Anything, "user-123").Return(user, nil)

	mockLobby.On("RequestAddAndRunRoom", mock.Anything, mock.AnythingOfType("*game.room")).Run(func(args mock.Arguments) {
		r := args.Get(1).(Room)
		desc := r.Description()
		assert.Equal(t, 4, desc.maxPlayers)
		assert.True(t, desc.private)
	}).Return()

	handler := NewGameHandler(mockLobby, mockUserGetter, mockWordGen)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/create", func(c *gin.Context) {
		c.Set("id", "user-123")
		handler.CreateGameHandler(c)
	})

	server := httptest.NewServer(router)
	defer server.Close()

	client := &http.Client{}

	// CHANGED: Query params instead of JSON body
	query := "maxPlayers=4&roundsCount=3&wordsCount=3&choosingWordDuration=30&drawingDuration=60&private=true"
	req, _ := http.NewRequest("GET", server.URL+"/create?"+query, nil)

	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)

	mockUserGetter.AssertExpectations(t)
	mockLobby.AssertExpectations(t)
}

func TestJoinGameHandler_Success(t *testing.T) {
	mockLobby := &MockLobby{}
	mockUserGetter := &MockUserGetter{}
	mockWordGen := &MockRandomWordsGenerator{}

	user := domain.User{Id: "user-456", Username: "JoinerPlayer"}
	mockUserGetter.On("GetUserById", mock.Anything, "user-456").Return(user, nil)

	mockLobby.On("ForwardPlayerJoinRequestToRoom", mock.Anything, mock.AnythingOfType("game.roomJoinRequest")).Run(func(args mock.Arguments) {
		req := args.Get(1).(roomJoinRequest)

		mockRoom := &MockRoom{}
		mockRoom.On("RemoveMe", mock.Anything, mock.Anything).Return()
		req.player.SetRoom(mockRoom)

		close(req.errChan)
	}).Return()

	handler := NewGameHandler(mockLobby, mockUserGetter, mockWordGen)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/join/:roomid", func(c *gin.Context) {
		c.Set("id", "user-456")
		handler.JoinGameHandler(c)
	})

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/join/ROOM-101"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	if conn != nil {
		defer conn.Close()
	}

	err = conn.WriteMessage(websocket.PingMessage, []byte{})
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	mockUserGetter.AssertExpectations(t)
	mockLobby.AssertExpectations(t)
}

func TestRealGameFlow_Integration(t *testing.T) {
	// 1. Setup Dependencies
	mockUserGetter := &MockUserGetter{}
	mockUserGetter.On("GetUserById", mock.Anything, "host-id").Return(domain.User{Id: "host-id", Username: "HostPlayer"}, nil)
	mockUserGetter.On("GetUserById", mock.Anything, "joiner-id").Return(domain.User{Id: "joiner-id", Username: "JoinerPlayer"}, nil)

	mockIdGen := &MockUniqueIdGenerator{}
	mockIdGen.On("Generate").Return("TEST-ROOM")
	mockIdGen.On("Dispose", "TEST-ROOM").Return()

	tickerGen := NewTickerGen()
	lobby := NewLobby(mockIdGen, &tickerGen)

	started := make(chan struct{})
	go lobby.LobbyActor(started)
	<-started

	gin.SetMode(gin.TestMode)
	handler := NewGameHandler(lobby, mockUserGetter, &MockRandomWordsGenerator{})
	router := gin.New()

	router.GET("/create", func(c *gin.Context) {
		c.Set("id", "host-id")
		handler.CreateGameHandler(c)
	})
	router.GET("/join/:roomid", func(c *gin.Context) {
		c.Set("id", "joiner-id")
		handler.JoinGameHandler(c)
	})

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/create"
	q := url.Values{}
	q.Add("maxPlayers", "4")
	q.Add("roundsCount", "3")
	q.Add("wordsCount", "3")
	q.Add("choosingWordDuration", "30")
	q.Add("drawingDuration", "60")
	q.Add("private", "false")

	hostWs, _, err := websocket.DefaultDialer.Dial(wsURL+"?"+q.Encode(), nil)
	assert.NoError(t, err)
	defer hostWs.Close()

	joinURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/join/TEST-ROOM"
	joinWs, _, err := websocket.DefaultDialer.Dial(joinURL, nil)
	assert.NoError(t, err)
	defer joinWs.Close()

	_, msg, err := joinWs.ReadMessage()
	assert.NoError(t, err)
	assert.Contains(t, string(msg), "TEST-ROOM")

	_, msg, err = hostWs.ReadMessage()
	assert.NoError(t, err)
	assert.Contains(t, string(msg), "TEST-ROOM")

	_, msg, err = hostWs.ReadMessage()
	assert.NoError(t, err)
	assert.Contains(t, string(msg), "JoinerPlayer")

	mockUserGetter.AssertExpectations(t)
}

func TestGameHandler_GetPublicGamesHandler(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	t.Run("success with multiple games", func(t *testing.T) {
		t.Parallel()

		mockLobby := &MockLobby{}
		mockUserGetter := &MockUserGetter{}
		mockWordGen := &MockRandomWordsGenerator{}

		userID := "user-123"
		mockUserGetter.On("GetUserById", mock.Anything, userID).Return(domain.User{Id: userID, Username: "TestUser"}, nil)

		expectedGames := []roomDescription{
			{id: "room-1", private: false, playersCount: 3, maxPlayers: 5, started: false},
			{id: "room-2", private: false, playersCount: 5, maxPlayers: 5, started: true},
		}

		mockLobby.On("GetPublicGames", mock.Anything).Return(expectedGames)

		handler := NewGameHandler(mockLobby, mockUserGetter, mockWordGen)

		router := gin.New()
		router.GET("/games", func(c *gin.Context) {
			c.Set("id", userID)
			handler.GetPublicGamesHandler(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/games", nil)
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)

		assert.Equal(t, http.StatusOK, res.Code)
		assert.Contains(t, res.Body.String(), `"id":"room-1"`)
		assert.Contains(t, res.Body.String(), `"playersCount":3`)
		assert.Contains(t, res.Body.String(), `"maxPlayers":5`)
		assert.Contains(t, res.Body.String(), `"started":false`)
		assert.Contains(t, res.Body.String(), `"id":"room-2"`)
		assert.Contains(t, res.Body.String(), `"started":true`)

		mockLobby.AssertExpectations(t)
	})

	t.Run("success with no games", func(t *testing.T) {
		t.Parallel()

		mockLobby := &MockLobby{}
		mockUserGetter := &MockUserGetter{}
		mockWordGen := &MockRandomWordsGenerator{}

		userID := "user-123"
		mockUserGetter.On("GetUserById", mock.Anything, userID).Return(domain.User{Id: userID, Username: "TestUser"}, nil)

		mockLobby.On("GetPublicGames", mock.Anything).Return([]roomDescription{})

		handler := NewGameHandler(mockLobby, mockUserGetter, mockWordGen)

		router := gin.New()
		router.GET("/games", func(c *gin.Context) {
			c.Set("id", userID)
			handler.GetPublicGamesHandler(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/games", nil)
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)

		assert.Equal(t, http.StatusOK, res.Code)
		assert.Equal(t, "[]", res.Body.String())

		mockLobby.AssertExpectations(t)
	})
}
