package game

import (
	"api/internal/shared/logger"
	"sync"
)

type MatchmakingQueue struct {
	heap   []*GameRoom
	locker sync.RWMutex
}

func (mq *MatchmakingQueue) Lock() {
	logger.Infof("[Matchmaking] LOCK acquired")
	mq.locker.Lock()
}

func (mq *MatchmakingQueue) Unlock() {
	logger.Infof("[Matchmaking] LOCK released")
	mq.locker.Unlock()
}

func (mq *MatchmakingQueue) RLock() {
	logger.Infof("[Matchmaking] RLOCK acquired")
	mq.locker.RLock()
}

func (mq *MatchmakingQueue) RUnlock() {
	logger.Infof("[Matchmaking] RLOCK released")
	mq.locker.RUnlock()
}

func (mq *MatchmakingQueue) games() []*GameRoom {
	return mq.heap
}

// TODO: OPTIMIZE set the matchmaking index only at the end
// Log logic: monitoring how rooms bubble up (usually when a player leaves or a new room is created)
func (mq *MatchmakingQueue) rebalanceAfterRemovingPlayer(index int) {
	if index < 0 || index >= len(mq.heap) {
		logger.Criticalf("[Matchmaking] CRITICAL: Attempted rebalance UP on invalid index: %d (Heap Size: %d)", index, len(mq.heap))
		return
	}

	logger.Infof("[Matchmaking] Rebalancing UP (Bubbling Up) starting from index %d. Player count: %d", index, mq.heap[index].numPlayers())

	parent := (index - 1) / 2

	for index > 0 && mq.heap[parent].numPlayers() > mq.heap[index].numPlayers() {
		logger.Infof("[Matchmaking] Swapping Index %d (%d players) with Parent %d (%d players)",
			index, mq.heap[index].numPlayers(), parent, mq.heap[parent].numPlayers())

		mq.heap[parent], mq.heap[index] = mq.heap[index], mq.heap[parent]
		mq.heap[parent].setMatchmakingIndex(parent)
		mq.heap[index].setMatchmakingIndex(index)

		index = parent
		parent = (index - 1) / 2
	}
	logger.Infof("[Matchmaking] Rebalance UP finished. Item settled at index %d", index)
}

// Log logic: monitoring how rooms bubble down (usually when a player joins)
func (mq *MatchmakingQueue) rebalanceAfterAddingPlayer(index int) {
	if index < 0 || index >= len(mq.heap) {
		logger.Criticalf("[Matchmaking] CRITICAL: Attempted rebalance DOWN on invalid index: %d (Heap Size: %d)", index, len(mq.heap))
		return
	}

	logger.Infof("[Matchmaking] Rebalancing DOWN (Bubbling Down) starting from index %d. Player count: %d", index, mq.heap[index].numPlayers())

	n := len(mq.heap)

	for {
		left := 2*index + 1
		right := 2*index + 2

		smallest := index
		if left < n && mq.heap[left].numPlayers() < mq.heap[smallest].numPlayers() {
			smallest = left
		}
		if right < n && mq.heap[right].numPlayers() < mq.heap[smallest].numPlayers() {
			smallest = right
		}
		if smallest == index {
			break
		}

		logger.Infof("[Matchmaking] Swapping Index %d (%d players) with Child %d (%d players)",
			index, mq.heap[index].numPlayers(), smallest, mq.heap[smallest].numPlayers())

		mq.heap[index], mq.heap[smallest] = mq.heap[smallest], mq.heap[index]

		mq.heap[index].setMatchmakingIndex(index)
		mq.heap[smallest].setMatchmakingIndex(smallest)
		index = smallest
	}
	logger.Infof("[Matchmaking] Rebalance DOWN finished. Item settled at index %d", index)
}

func (mq *MatchmakingQueue) assignPlayer(p *Player) {
	logger.Infof("[Matchmaking] Request to assign player %s. Current Heap Size: %d", p.username, len(mq.heap))

	if len(mq.heap) == 0 || mq.heap[0].numPlayers() >= MAX_PLAYERS {
		if len(mq.heap) > 0 {
			logger.Infof("[Matchmaking] Top room (ID: %s) is full (%d/%d). Creating new room.", mq.heap[0].id, mq.heap[0].numPlayers(), MAX_PLAYERS)
		} else {
			logger.Infof("[Matchmaking] Heap is empty. Creating first room.")
		}

		newroom := newGameroom()
		newroom.addPlayer(p)
		mq.heap = append(mq.heap, newroom)

		newIndex := len(mq.heap) - 1
		newroom.setMatchmakingIndex(newIndex)

		logger.Infof("[Matchmaking] New room %s created at index %d. Triggering Rebalance UP.", newroom.id, newIndex)
		mq.rebalanceAfterRemovingPlayer(newIndex) // Uses "Remove" logic because new room has few players, needs to bubble UP
	} else {
		targetRoom := mq.heap[0]
		logger.Infof("[Matchmaking] Assigning to existing top room %s (Current Players: %d)", targetRoom.id, targetRoom.numPlayers())

		targetRoom.Lock()
		targetRoom.addPlayer(p)
		targetRoom.Unlock()

		logger.Infof("[Matchmaking] Player added. Triggering Rebalance DOWN from top.")
		mq.rebalanceAfterAddingPlayer(targetRoom.matchmakingIndex) // Uses "Add" logic because top room got bigger, might need to bubble DOWN
	}

	logger.Infof("[Matchmaking] Player %s successfully assigned to room %s", p.username, p.room.id)
}

func (mq *MatchmakingQueue) assignPlayerTo(p *Player, gr *GameRoom) error {
	logger.Infof("[Matchmaking] Direct assignment of %s to room %s (Index: %d)", p.username, gr.id, gr.matchmakingIndex)
	gr.Lock()
	err := gr.addPlayer(p)
	gr.Unlock()
	if err != nil {
		return err
	}
	mq.rebalanceAfterAddingPlayer(gr.matchmakingIndex)
	return nil
}

// func (mq *MatchmakingQueue) removeGame(gr *GameRoom) {
//  logger.Infof("[Matchmaking] Removing gameroom %s", gr.id)
//  index := gr.matchmakingIndex

//  if index == -1 {
//      return
//  }

//  if len(mq.heap) <= 1 {
//      mq.heap = make([]*GameRoom, 0, 100)
//  } else if index == len(mq.heap)-1 {
//      mq.heap = mq.heap[:len(mq.heap)-1]
//  } else {
//      mq.heap[index] = mq.heap[len(mq.heap)-1]
//      mq.heap = mq.heap[:len(mq.heap)-1]
//      mq.rebalanceAfterAddingPlayer(index)
//  }

//  gr.matchmakingIndex = -1
//  logger.Infof("[Matchmaking] Gameroom %s removed", gr.id)
// }

func (mq *MatchmakingQueue) removeGame(gr *GameRoom) {

	logger.Infof("[Matchmaking] Removing gameroom %s (Current Heap Index: %d)", gr.id, gr.matchmakingIndex)
	index := gr.matchmakingIndex

	if index == -1 {
		logger.Warningf("[Matchmaking] Attempted to remove game %s but index is -1. Already removed?", gr.id)
		return
	}

	if index >= len(mq.heap) {
		logger.Criticalf("[Matchmaking] CRITICAL: Game %s has index %d but heap size is %d", gr.id, index, len(mq.heap))
		return
	}

	if len(mq.heap) <= 1 {
		logger.Infof("[Matchmaking] Heap was size 1 or 0. Resetting heap.")
		mq.heap = make([]*GameRoom, 0, 100)
	} else if index == len(mq.heap)-1 {
		logger.Infof("[Matchmaking] Removing last element. No rebalance needed.")
		mq.heap = mq.heap[:len(mq.heap)-1]
	} else {
		lastIndex := len(mq.heap) - 1
		logger.Infof("[Matchmaking] Swapping removed index %d with last index %d", index, lastIndex)

		mq.heap[index] = mq.heap[lastIndex]
		mq.heap = mq.heap[:lastIndex]

		// Update the index of the moved room
		mq.heap[index].setMatchmakingIndex(index)
		logger.Infof("[Matchmaking] Triggering Rebalance DOWN for moved node at index %d", index)
		mq.rebalanceAfterAddingPlayer(index)
		mq.rebalanceAfterRemovingPlayer(index)
	}

	gr.matchmakingIndex = -1
	logger.Infof("[Matchmaking] Gameroom %s removed successfully. New Heap Size: %d", gr.id, len(mq.heap))
}

var matchmaking MatchmakingQueue
