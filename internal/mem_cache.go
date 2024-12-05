package internal

import (
	"github.com/patrickmn/go-cache"
	logger "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var Cache *MemCache

type MemCache struct {
	*cache.Cache
	localFilePath string
}

func NewClient(localFile string) *MemCache {
	memCache := cache.New(24*time.Hour, 15*time.Minute)
	// Wait for SIGINT (interrupt) signal.
	m := &MemCache{Cache: memCache, localFilePath: localFile}
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		for {
			select {
			case <-shutdownChan:
				signal.Stop(shutdownChan)
				m.Close()
				return
			}
		}
	}()
	return m
}

func (m *MemCache) Close() {
	if m.localFilePath != "" {
		err := m.SaveFile(m.localFilePath)
		if err != nil {
			logger.Errorf("cache save file err: %v", err)
		}
	}

}

func init() {
	localFile := ""
	if Config.Server.CacheFile != "" {
		localFile = GetProcessPath() + "/" + Config.Server.CacheFile
	}
	Cache = NewClient(localFile)
}
