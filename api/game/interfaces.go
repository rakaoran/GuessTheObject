package game

import (
	"api/domain"
	"context"
)

type WebsocketConnection interface {
	Close(errCode string)
	Write(data []byte) error
	Read() ([]byte, error)
	Ping() error
}

type UserGetter interface {
	GetUserById(ctx context.Context, id string) (domain.User, error)
}
