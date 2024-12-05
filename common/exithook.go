package common

import (
	"fmt"
	"github.com/hefeiyu2025/pan-client/internal"
	logger "github.com/sirupsen/logrus"
	"os"
)

func init() {
	go func() {
		select {
		case <-internal.ExitChan:
			fmt.Print("Exit")
			internal.WaitGroup.Add(1)
			internal.CacheExitChan <- struct{}{}
			internal.WaitGroup.Done()
			return
		}
	}()
}

func Exit() {
	internal.WaitGroup.Add(1)
	internal.ExitChan <- struct{}{}
	internal.WaitGroup.Wait()
	if r := recover(); r != nil {
		logger.Error(r)
		os.Exit(1)
	}
	os.Exit(0)
}
