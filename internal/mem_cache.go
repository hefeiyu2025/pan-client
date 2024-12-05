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

var CacheExitChan = make(chan struct{}, 1)

type MemCache struct {
	*cache.Cache
	localFilePath string
}

func NewClient(localFile string) *MemCache {
	memCache := cache.New(12*time.Hour, 30*time.Minute)
	// Wait for SIGINT (interrupt) signal.
	m := &MemCache{Cache: memCache, localFilePath: localFile}
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		for {
			select {
			case <-shutdownChan:
				signal.Stop(shutdownChan)
				m.save()
				return
			case <-CacheExitChan:
				m.save()
				WaitGroup.Done()
				return
			}
		}
	}()
	m.load()
	return m
}
func (m *MemCache) load() {
	if m.localFilePath != "" {
		if _, err := os.Stat(m.localFilePath); os.IsNotExist(err) {
			logger.Errorf("cache load file %s err: %v", m.localFilePath, err)
			return
		}
		err := m.LoadFile(m.localFilePath)
		if err != nil {
			logger.Errorf("cache load file %s err: %v", m.localFilePath, err)
		}
	}
}

func (m *MemCache) save() {
	if m.localFilePath != "" {
		m.DeleteExpired()
		err := m.SaveFile(m.localFilePath)
		if err != nil {
			logger.Errorf("cache save file err: %v", err)
		}
	}
}

func InitCache() {
	localFile := ""
	if Config.Server.CacheFile != "" {
		localFile = GetProcessPath() + "/" + Config.Server.CacheFile
	}
	Cache = NewClient(localFile)
}
