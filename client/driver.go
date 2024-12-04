package client

import (
	"fmt"
	"github.com/hefeiyu2025/pan-client/internal"
	_ "github.com/hefeiyu2025/pan-client/internal"
	"github.com/spf13/viper"
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
	Rename()
	Mkdir()
	Move()
	Delete()
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
	c.SetDuration(key, value, -1)
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

type DriverErrorInterface interface {
	GetCode() int
	GetMsg() string
	GetErr() error
	GetData() interface{}
	Error() string
}

// DriverError 定义全局的基础异常
type DriverError struct {
	Code int
	Msg  string
	Err  error
	Data interface{}
}

func (e *DriverError) GetCode() int {
	return e.Code
}

func (e *DriverError) GetMsg() string {
	return e.Msg
}

func (e *DriverError) GetErr() error {
	return e.Err
}

func (e *DriverError) GetData() interface{} {
	return e.Data
}

func (e *DriverError) Error() string {
	m := make(map[string]any)
	m["code"] = e.Code
	m["msg"] = e.Msg
	m["err"] = e.Err
	m["data"] = e.Data
	errorStr := ""
	for key, value := range m {
		if value != nil {
			errorStr = e.appendKeyValue(errorStr, key, value)
		}
	}
	return errorStr
}

func (e *DriverError) appendKeyValue(errorStr, key string, value interface{}) string {
	errorStr = errorStr + key
	errorStr = errorStr + "="
	stringVal, ok := value.(string)
	if !ok {
		stringVal = fmt.Sprint(value)
	}
	return errorStr + fmt.Sprintf("%q", stringVal) + " "
}
