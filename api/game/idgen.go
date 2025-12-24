package game

import "sync"

type Idgen struct {
	idsBySize        map[int]map[string]struct{}
	minAvailableSize int
	locker           sync.RWMutex
}

func (idgen *Idgen) GetId() {
	idgen.locker.Lock()

}
