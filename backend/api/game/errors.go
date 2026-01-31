package game

import "errors"

var (
	ErrRoomNotFound = errors.New("room-not-found")
	ErrRoomFull     = errors.New("room-full")
)

var ErrSendBufferFull = errors.New("send-buffer-full")
