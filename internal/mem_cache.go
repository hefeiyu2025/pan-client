package internal

import (
	"github.com/patrickmn/go-cache"
	logger "github.com/sirupsen/logrus"
	"os"
	"time"
)

var Cache *MemCache

type MemCache struct {
	*cache.Cache
	localFilePath string
}

func NewClient(localFile string) *MemCache {
	memCache := cache.New(12*time.Hour, 30*time.Minute)
	// Wait for SIGINT (interrupt) signal.
	m := &MemCache{Cache: memCache, localFilePath: localFile}

	go func() {
		select {
		case <-CacheExitChan:
			m.save()
			ExitWaitGroup.Done()
		}
	}()
	m.load()
	return m
}
func (m *MemCache) load() {
	if m.localFilePath != "" {
		if _, err := os.Stat(m.localFilePath); err != nil {
			if !os.IsNotExist(err) {
				logger.Errorf("cache load file %s err: %v", m.localFilePath, err)
				return
			}
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
		logger.Infof("cache save file %s success", m.localFilePath)
	}
}

func InitCache() {
	localFile := ""
	if Config.Server.CacheFile != "" {
		localFile = GetProcessPath() + "/" + Config.Server.CacheFile
	}
	Cache = NewClient(localFile)
}
