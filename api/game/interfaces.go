package game

type NetworkSession interface {
	Close(errCode string)
	Write(data []byte) error
	Read() ([]byte, error)
	Ping() error
}
