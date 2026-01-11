package game

import (
	"context"
	"sync"
)

type service struct {
	locker     sync.RWMutex
	rooms      map[string]*Room
	idGen      Idgen
	userGetter UserGetter
}

func NewService(userGetter UserGetter) service {
	return service{
		rooms:      make(map[string]*Room),
		userGetter: userGetter,
		idGen:      NewIdGen(),
	}
}

func (s *service) CreateRoom(ctx context.Context, playerId string, playerWS WebsocketConnection, private bool, configs RoomConfigs) error {
	user, err := s.userGetter.GetUserById(ctx, playerId)

	if err != nil {
		playerWS.Close("unknown error")
		return err
	}
	roomId := s.idGen.GetUniqueId()
	player := NewPlayer(playerId, user.Username, playerWS)
	room := NewRoom(roomId, player, configs, private)
	go player.ReadPump()
	go player.WritePump()
	s.locker.Lock()
	s.rooms[roomId] = room
	s.locker.Unlock()

	// TODO: Run game loop here
	playerWS.Close("")
	return nil
}

func (s *service) JoinRoom(ctx context.Context, playerId, roomId string, playerWS WebsocketConnection) error {
	s.locker.RLock()
	room, exists := s.rooms[roomId]
	s.locker.RUnlock()

	user, err := s.userGetter.GetUserById(ctx, playerId)

	player := NewPlayer(playerId, user.Username, playerWS)

	if err != nil {
		playerWS.Close("unknown error")
		return err
	}

	if !exists {
		return ErrRoomNotFound
	}

}
