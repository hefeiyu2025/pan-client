package internal

import "sync"

var ExitChan = make(chan struct{}, 1)
var WaitGroup sync.WaitGroup
