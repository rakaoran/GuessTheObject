package game

import "sync"

type Idgen struct {
	idsBySize        map[int]map[string]struct{}
	minAvailableSize int
	locker           sync.RWMutex
}

func (idgen *Idgen) GetPublicId() {
	idgen.locker.Lock()

}
