package game

type NetworkSession interface {
	close(errCode string)
	send(data []byte) error
}
