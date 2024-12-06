package client

import (
	"fmt"
	"github.com/hefeiyu2025/pan-client/internal"
	logger "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"os"
	"path/filepath"
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

type BaseOperate struct {
}

func (b *BaseOperate) BaseUploadPath(req OneStepUploadPathReq, UploadFile func(req OneStepUploadFileReq) error) error {
	localPath := req.LocalPath
	if localPath != "" {
		fileInfo, err := os.Stat(localPath)
		if err != nil {
			logger.Errorf("file %s read error %v", localPath, err)
			return OnlyError(err)
		}
		if !fileInfo.IsDir() {
			err = UploadFile(OneStepUploadFileReq{
				LocalFile:      localPath,
				RemotePath:     req.RemotePath,
				Resumable:      req.Resumable,
				SuccessDel:     req.SuccessDel,
				RemoteTransfer: req.RemoteTransfer,
			})
			return err
		}
		return filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				for _, ignorePath := range req.IgnorePaths {
					if filepath.Base(path) == ignorePath {
						return filepath.SkipDir
					}
				}
			} else {
				// 获取相对于root的相对路径
				relPath, _ := filepath.Rel(localPath, path)
				relPath = strings.Replace(relPath, "\\", "/", -1)
				relPath = strings.Replace(relPath, info.Name(), "", 1)
				NotUpload := false
				for _, ignoreFile := range req.IgnoreFiles {
					if info.Name() == ignoreFile {
						NotUpload = true
						break
					}
				}
				for _, extension := range req.IgnoreExtensions {
					if strings.HasSuffix(info.Name(), extension) {
						NotUpload = true
						break
					}
				}
				for _, extension := range req.Extensions {
					if strings.HasSuffix(info.Name(), extension) {
						NotUpload = false
						break
					}
					NotUpload = true
				}
				if !NotUpload {
					err = UploadFile(OneStepUploadFileReq{
						LocalFile:      path,
						RemotePath:     strings.TrimRight(req.RemotePath, "/") + "/" + relPath,
						Resumable:      req.Resumable,
						SuccessDel:     req.SuccessDel,
						RemoteTransfer: req.RemoteTransfer,
					})
					if err == nil {
						if req.SuccessDel {
							dir := filepath.Dir(path)
							if dir != "." {
								empty, _ := internal.IsEmptyDir(dir)
								if empty {
									_ = os.Remove(dir)
									fmt.Println("uploaded success and delete", dir)
								}
							}
						}
					} else {
						if !req.SkipFileErr {
							return err
						} else {
							logger.Errorf("upload err %v", err)
						}
					}
				}
			}
			return nil
		})
	}
	// 遍历目录
	return OnlyMsg("path is empty")
}

func (b *BaseOperate) BaseDownloadPath(req OneStepDownloadPathReq,
	List func(req ListReq) ([]*PanObj, error),
	DownloadFile func(req OneStepDownloadFileReq) error) error {
	dir := req.RemotePath
	logger.Infof("start download dir %s", strings.Trim(dir.Path, "/")+"/"+dir.Name)
	if dir.Type != "dir" {
		return OnlyMsg("only support download dir")
	}
	objs, err := List(ListReq{
		Reload: true,
		Dir:    req.RemotePath,
	})
	if err != nil {
		return err
	}
	for _, object := range objs {
		if object.Type == "dir" {
			err = b.BaseDownloadPath(OneStepDownloadPathReq{
				RemotePath:       object,
				LocalPath:        req.LocalPath,
				Concurrency:      req.Concurrency,
				ChunkSize:        req.ChunkSize,
				DownloadCallback: req.DownloadCallback,
			}, List, DownloadFile)
			if err != nil {
				return err
			}
		} else {
			err = DownloadFile(OneStepDownloadFileReq{
				RemoteFile:       object,
				LocalPath:        req.LocalPath,
				Concurrency:      req.Concurrency,
				ChunkSize:        req.ChunkSize,
				DownloadCallback: req.DownloadCallback,
			})
			if err != nil {
				return err
			}
		}
	}
	fmt.Println("end download dir", strings.Trim(dir.Path, "/")+"/"+dir.Name)
	return nil
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
	return internal.Cache.Get(string(c.DriverType) + "." + key)
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

type ProgressReader struct {
	readCloser      io.ReadCloser
	file            *os.File
	uploaded        int64
	chunkSize       int64
	totalSize       int64
	currentSize     int64
	currentUploaded int64
	currentChunkNum int64
	finish          bool
	startTime       time.Time
}

func (pr *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = pr.readCloser.Read(p)
	if n > 0 {
		pr.currentUploaded += int64(n)
		elapsed := time.Since(pr.startTime).Seconds()
		var speed float64
		if elapsed == 0 {
			speed = float64(pr.currentUploaded) / 1024
		} else {
			speed = float64(pr.currentUploaded) / 1024 / elapsed // KB/s
		}

		// 计算进度百分比
		percent := float64(pr.uploaded+pr.currentUploaded) / float64(pr.totalSize) * 100
		logger.Infof("\ruploading: %.2f%% (%d/%d bytes, %.2f KB/s)", percent, pr.uploaded+pr.currentUploaded, pr.totalSize, speed)
		// 相等即已经处理完毕
		if pr.currentSize == pr.currentUploaded {
			pr.uploaded += pr.currentSize
		}
		if pr.uploaded == pr.totalSize {
			pr.finish = true
		}
	}
	return n, err
}
func (pr *ProgressReader) NextChunk() (int64, int64) {
	pr.readCloser = io.NopCloser(&io.LimitedReader{
		R: pr.file,
		N: pr.chunkSize,
	})
	startSize := pr.uploaded
	endSize := min(pr.totalSize, pr.uploaded+pr.chunkSize)
	pr.currentSize = endSize
	return startSize, endSize
}

func (pr *ProgressReader) Close() {
	name := pr.file.Name()
	if pr.file != nil {
		pr.file.Close()
	}
	logger.Infof("file:%s uploaded:%d,total:%d,%.2f%%", name, pr.uploaded, pr.totalSize, float64(pr.uploaded)/float64(pr.totalSize)*100)
}
func (pr *ProgressReader) GetTotal() int64 {
	return pr.totalSize
}

func (pr *ProgressReader) GetUploaded() int64 {
	return pr.uploaded
}

func (pr *ProgressReader) IsFinish() bool {
	return pr.finish
}

func NewProcessReader(localFile string, chunkSize, uploaded int64) (*ProgressReader, DriverErrorInterface) {
	file, err := os.Open(localFile)
	if err != nil {
		return nil, OnlyError(err)
	}
	stat, err := file.Stat()
	if err != nil {
		return nil, OnlyError(err)
	}
	// 判断是否目录，目录则无法处理
	if stat.IsDir() {
		return nil, OnlyMsg(localFile + " not a file")
	}
	// 计算剩余字节数
	totalSize := stat.Size()
	leftSize := totalSize - uploaded

	chunkNum := (leftSize / chunkSize) + 1
	if uploaded > 0 {
		// 将文件指针移动到指定的分片位置
		ret, _ := file.Seek(uploaded, 0)
		if ret == 0 {
			return nil, OnlyMsg(localFile + " seek file failed")
		}
	}
	return &ProgressReader{
		file:            file,
		uploaded:        uploaded,
		chunkSize:       chunkSize,
		totalSize:       totalSize,
		currentChunkNum: chunkNum,
		startTime:       time.Now(),
	}, nil
}
