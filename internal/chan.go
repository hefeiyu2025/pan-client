package internal

import "sync"

var ExitChan = make(chan struct{})
var ChunkExitChan = make(chan struct{})
var CacheExitChan = make(chan struct{})
var ExitWaitGroup sync.WaitGroup
var ExitChanList = []chan struct{}{
	ChunkExitChan, CacheExitChan,
}
