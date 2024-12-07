package common

import (
	"github.com/hefeiyu2025/pan-client/internal"
	logger "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

func InitExitHook() {
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-shutdownChan:
			logger.Info("person exit hook")
			signal.Stop(shutdownChan)
			for _, c := range internal.ExitChanList {
				internal.ExitWaitGroup.Add(1)
				c <- struct{}{}
			}
			internal.ExitWaitGroup.Wait()
			logger.Info("person exit hook finish")
			os.Exit(2)
		case <-internal.ExitChan:
			logger.Info("auto exit hook")
			for _, c := range internal.ExitChanList {
				internal.ExitWaitGroup.Add(1)
				c <- struct{}{}
			}
			internal.ExitWaitGroup.Done()
			logger.Info("auto exit hook finish")
			return
		}
	}()
}

func Exit() {
	internal.ExitWaitGroup.Add(1)
	internal.ExitChan <- struct{}{}
	internal.ExitWaitGroup.Wait()
	if r := recover(); r != nil {
		logger.Error(r)
		os.Exit(1)
	}
	os.Exit(0)
}
