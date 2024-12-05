package client

import (
	"fmt"
	_ "github.com/hefeiyu2025/pan-client/common"
	"github.com/hefeiyu2025/pan-client/internal"
	"github.com/spf13/viper"
	"strings"
	"sync"
	"time"
)

type DriverConstructor func() Driver

type DriverType string

const (
	ViperDriverPrefix string     = "driver."
	Cloudreve         DriverType = "cloudreve"
)

type Driver interface {
	Meta
	Operate
	Share
}

type Meta interface {
	Init() error
	Drop() error
	ReadConfig(config Properties) error
	WriteConfig(config Properties) error
	Get(key string) (interface{}, bool)
	GetOrDefault(key string, defFun DefaultFun) (interface{}, bool, error)
	Set(key string, value interface{})
	SetDuration(key string, value interface{}, d time.Duration)
	Del(key string)
}

type Operate interface {
	Disk() (*DiskResp, error)
	List(req ListReq) ([]*PanObj, error)
	ObjRename(req ObjRenameReq) error
	BatchRename(req BatchRenameReq) error
	Mkdir(req MkdirReq) (*PanObj, error)
	Move(req MovieReq) error
	Delete(req DeleteReq) error
	UploadPath(req OneStepUploadPathReq) error
	UploadFile(req OneStepUploadFileReq) error
	DownloadPath(req OneStepDownloadPathReq) error
	DownloadFile(req OneStepDownloadFileReq) error
}

type Share interface {
	ShareList()
	NewShare()
	DeleteShare()
}

type PropertiesOperate struct {
	DriverType DriverType
	m          sync.RWMutex
}

func (c *PropertiesOperate) ReadConfig(config Properties) error {
	c.m.RLock()
	defer c.m.RUnlock()
	internal.SetDefaultByTag(config)
	return viper.UnmarshalKey(ViperDriverPrefix+string(c.DriverType), config)
}

func (c *PropertiesOperate) WriteConfig(config Properties) error {
	c.m.Lock()
	defer c.m.Unlock()
	viper.Set(ViperDriverPrefix+string(c.DriverType), config)
	return viper.WriteConfig()
}

type CacheOperate struct {
	DriverType DriverType
	w          sync.Mutex
}

func (c *CacheOperate) Get(key string) (interface{}, bool) {
	return internal.Cache.Get(key)
}

type DefaultFun func() (interface{}, error)

func (c *CacheOperate) GetOrDefault(key string, defFun DefaultFun) (interface{}, bool, error) {
	result, ok := internal.Cache.Get(string(c.DriverType) + "." + key)
	if !ok {
		r, err := defFun()
		if err != nil {
			return nil, false, err
		}
		c.Set(key, r)
		result = r
	}
	return result, true, nil
}

func (c *CacheOperate) Set(key string, value interface{}) {
	c.w.Lock()
	defer c.w.Unlock()
	internal.Cache.SetDefault(string(c.DriverType)+"."+key, value)
}

func (c *CacheOperate) SetDuration(key string, value interface{}, d time.Duration) {
	c.w.Lock()
	defer c.w.Unlock()
	internal.Cache.Set(string(c.DriverType)+"."+key, value, d)
}

func (c *CacheOperate) Del(key string) {
	c.w.Lock()
	defer c.w.Unlock()
	internal.Cache.Delete(string(c.DriverType) + "." + key)
}

type CommonOperate struct {
}

func (c *CommonOperate) GetPanObj(path string, mustExist bool, list func(req ListReq) ([]*PanObj, error)) (*PanObj, error) {
	truePath := strings.Trim(path, "/")
	paths := strings.Split(truePath, "/")

	target := &PanObj{
		Id:   "0",
		Name: "",
		Path: "/",
		Size: 0,
		Type: "dir",
	}
	for _, pathStr := range paths {
		if pathStr == "" {
			continue
		}
		currentChildren, err := list(ListReq{
			Reload: false,
			Dir:    target,
		})
		if err != nil {
			return nil, OnlyError(err)
		}
		exist := false
		for _, file := range currentChildren {
			if file.Name == pathStr {
				target = file
				exist = true
				break
			}
		}
		// 如果必须存在且不存在，则返回错误
		if mustExist && !exist {
			return nil, OnlyMsg(fmt.Sprintf("%s not found", path))
		}

		// 如果不需要必须存在且不存在，则跳出循环
		if !exist {
			break
		}
	}
	return target, nil
}
