package common

import (
	"github.com/hefeiyu2025/pan-client/internal"
	logger "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

var isPersonShutdown = false
var personShutdownChan = make(chan struct{})

func InitExitHook() {
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-shutdownChan:
			signal.Stop(shutdownChan)
			isPersonShutdown = true
			for _, c := range internal.ExitChanList {
				internal.ExitWaitGroup.Add(1)
				c <- struct{}{}
			}
			internal.ExitWaitGroup.Wait()
			close(personShutdownChan)
		case <-internal.ExitChan:
			for _, c := range internal.ExitChanList {
				internal.ExitWaitGroup.Add(1)
				c <- struct{}{}
			}
			internal.ExitWaitGroup.Done()
			return
		}
	}()
}

func Exit() {
	if r := recover(); r != nil {
		logger.Error(r)
	}
	// 要是人工点击了关闭，那退出方法就无效了
	if isPersonShutdown {
		select {
		case <-personShutdownChan:
			return
		}
	}
	internal.ExitWaitGroup.Add(1)
	internal.ExitChan <- struct{}{}
	internal.ExitWaitGroup.Wait()
}
