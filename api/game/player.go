package game

type Player struct {
	id          string
	username    string
	score       int
	rateLimiter RateLimiter
	socket      NetworkSession
	inbox       ServerPacket
	pingChan    chan struct{}
	room        *Room
}

type RateLimiter struct {
}
