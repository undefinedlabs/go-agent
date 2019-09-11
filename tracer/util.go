package tracer

import (
	"go.undefinedlabs.com/scopeagent/ntp"
	"math/rand"
	"sync"
)

var (
	seededIDGen = rand.New(rand.NewSource(ntp.Now().UnixNano()))
	// The golang rand generators are *not* intrinsically thread-safe.
	seededIDLock sync.Mutex
)

func randomID() uint64 {
	seededIDLock.Lock()
	defer seededIDLock.Unlock()
	return uint64(seededIDGen.Int63())
}

func randomID2() (uint64, uint64) {
	seededIDLock.Lock()
	defer seededIDLock.Unlock()
	return uint64(seededIDGen.Int63()), uint64(seededIDGen.Int63())
}
