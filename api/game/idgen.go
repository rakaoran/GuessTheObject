package game

import (
	"math/rand"
)

var chars = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func NewIdGen() idgen {
	return idgen{
		ids: make(map[string]struct{}),
	}
}
func (idgen *idgen) GetUniqueId() string {
	idgen.locker.Lock()
	defer idgen.locker.Unlock()
	var id string
	for {
		r1 := rand.Intn(36)
		r2 := rand.Intn(36)
		r3 := rand.Intn(36)
		r4 := rand.Intn(36)
		r5 := rand.Intn(36)

		arr := []byte{chars[r1], chars[r2], chars[r3], chars[r4], chars[r5]}
		id = string(arr)

		if _, exists := idgen.ids[id]; !exists {
			idgen.ids[id] = struct{}{}
			break
		}
	}
	return id
}

func (idgen *idgen) DisposeId(id string) {
	idgen.locker.Lock()
	defer idgen.locker.Unlock()
	delete(idgen.ids, id)
}
