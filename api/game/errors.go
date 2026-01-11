package game

import "errors"

var (
	ErrRoomNotFound = errors.New("Room not found")
	ErrRoomFull     = errors.New("Room full")
)
