package pan

import (
	"fmt"
	"github.com/hefeiyu2025/pan-client/internal"
	"github.com/imroc/req/v3"
	logger "github.com/sirupsen/logrus"
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
	Quark             DriverType = "quark"
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
	UploadPath(req UploadPathReq) error
	UploadFile(req UploadFileReq) error
	DownloadPath(req DownloadPathReq) error
	DownloadFile(req DownloadFileReq) error
}

type BaseOperate struct {
}

func (b *BaseOperate) BaseUploadPath(req UploadPathReq, UploadFile func(req UploadFileReq) error) error {
	localPath := req.LocalPath
	if localPath != "" {
		fileInfo, err := os.Stat(localPath)
		if err != nil {
			logger.Errorf("file %s read error %v", localPath, err)
			return OnlyError(err)
		}
		if !fileInfo.IsDir() {
			err = UploadFile(UploadFileReq{
				LocalFile:          localPath,
				RemotePath:         req.RemotePath,
				Resumable:          req.Resumable,
				OnlyFast:           req.OnlyFast,
				SuccessDel:         req.SuccessDel,
				RemotePathTransfer: req.RemotePathTransfer,
				RemoteNameTransfer: req.RemotePathTransfer,
			})
			return err
		}
		logger.Infof("start upload dir %s -> %s", localPath, req.RemotePath)
		err = filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
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
				for _, extension := range req.Extensions {
					if strings.HasSuffix(info.Name(), extension) {
						NotUpload = false
						break
					}
					NotUpload = true
				}
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
				if !NotUpload {
					logger.Infof("start upload file %s -> %s", path, strings.TrimRight(req.RemotePath, "/")+"/"+relPath)
					err = UploadFile(UploadFileReq{
						LocalFile:          path,
						RemotePath:         strings.TrimRight(req.RemotePath, "/") + "/" + relPath,
						OnlyFast:           req.OnlyFast,
						Resumable:          req.Resumable,
						SuccessDel:         req.SuccessDel,
						RemotePathTransfer: req.RemotePathTransfer,
						RemoteNameTransfer: req.RemotePathTransfer,
					})
					if err == nil {
						dir := filepath.Dir(path)
						logger.Infof("uploaded success %s", dir)
						if req.SuccessDel {
							if dir != "." {
								empty, _ := internal.IsEmptyDir(dir)
								if empty {
									_ = os.Remove(dir)
									logger.Infof("delete success %s", dir)
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
					logger.Infof("end upload file %s -> %s", path, strings.TrimRight(req.RemotePath, "/")+"/"+relPath)
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		logger.Infof("end upload dir %s -> %s", localPath, req.RemotePath)
	}
	// 遍历目录
	return OnlyMsg("path is empty")
}

func (b *BaseOperate) BaseDownloadPath(req DownloadPathReq,
	List func(req ListReq) ([]*PanObj, error),
	DownloadFile func(req DownloadFileReq) error) error {
	dir := req.RemotePath
	remotePathName := strings.Trim(dir.Path, "/") + "/" + dir.Name
	logger.Infof("start download dir %s -> %s", remotePathName, req.LocalPath)
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
		NotDownload := false
		objectName := object.Name
		if req.RemoteNameTransfer != nil {
			objectName = req.RemoteNameTransfer(objectName)
		}
		if object.Type == "dir" {
			for _, ignorePath := range req.IgnorePaths {
				if objectName == ignorePath {
					NotDownload = true
					break
				}
			}
			if !NotDownload {
				err = b.BaseDownloadPath(DownloadPathReq{
					RemotePath:       object,
					LocalPath:        req.LocalPath,
					Concurrency:      req.Concurrency,
					ChunkSize:        req.ChunkSize,
					OverCover:        req.OverCover,
					DownloadCallback: req.DownloadCallback,
				}, List, DownloadFile)
				if err != nil {
					if req.SkipFileErr {
						logger.Errorf("download %s,err: %v", objectName, err)
					} else {
						return err
					}
				}
			} else {
				logger.Infof("dir will skip: %s", objectName)
			}
		} else {
			for _, extension := range req.Extensions {
				if strings.HasSuffix(objectName, extension) {
					NotDownload = false
					break
				}
				NotDownload = true
			}
			for _, ignoreFile := range req.IgnoreFiles {
				if objectName == ignoreFile {
					NotDownload = true
					break
				}
			}
			for _, extension := range req.IgnoreExtensions {
				if strings.HasSuffix(objectName, extension) {
					NotDownload = true
					break
				}
			}
			if !NotDownload {
				err = DownloadFile(DownloadFileReq{
					RemoteFile:       object,
					LocalPath:        req.LocalPath,
					Concurrency:      req.Concurrency,
					ChunkSize:        req.ChunkSize,
					OverCover:        req.OverCover,
					DownloadCallback: req.DownloadCallback,
				})
				if err != nil {
					if req.SkipFileErr {
						logger.Errorf("download %s,err: %v", objectName, err)
					} else {
						return err
					}
				}
			} else {
				logger.Infof("file will skip: %s", objectName)
			}
		}
	}
	logger.Infof("end download dir %s -> %s", remotePathName, req.LocalPath)
	return nil
}

type DownloadUrl func(req DownloadFileReq) (string, error)

func (b *BaseOperate) BaseDownloadFile(req DownloadFileReq,
	client *req.Client,
	downloadUrl DownloadUrl) error {
	object := req.RemoteFile
	if object.Type != "file" {
		return OnlyMsg("only support download file")
	}
	remoteFileName := strings.Trim(object.Path, "/") + "/" + object.Name
	logger.Infof("start download file %s", remoteFileName)
	outputFile := req.LocalPath + "/" + object.Name
	fileInfo, err := internal.IsExistFile(outputFile)
	if fileInfo != nil && err == nil {
		if fileInfo.Size() == object.Size && !req.OverCover {
			if req.DownloadCallback != nil {
				abs, _ := filepath.Abs(outputFile)
				req.DownloadCallback(filepath.Dir(abs), abs)
			}
			logger.Infof("end download file %s -> %s", remoteFileName, outputFile)
			return nil
		}
	}
	url, err := downloadUrl(req)
	if err != nil {
		return err
	}
	e := internal.NewChunkDownload(url, client).
		SetFileSize(object.Size).
		SetChunkSize(req.ChunkSize).
		SetConcurrency(req.Concurrency).
		SetOutputFile(outputFile).
		SetTempRootDir(internal.Config.Server.DownloadTmpPath).
		Do()
	if e != nil {
		logger.WithError(e).Errorf("error download file %s", remoteFileName)
		return e
	}

	logger.Infof("end download file %s -> %s", remoteFileName, outputFile)
	if req.DownloadCallback != nil {
		abs, _ := filepath.Abs(outputFile)
		req.DownloadCallback(filepath.Dir(abs), abs)
	}
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
	return internal.Viper.UnmarshalKey(ViperDriverPrefix+string(c.DriverType), config)
}

func (c *PropertiesOperate) WriteConfig(config Properties) error {
	c.m.Lock()
	defer c.m.Unlock()
	internal.Viper.Set(ViperDriverPrefix+string(c.DriverType), config)
	return internal.Viper.WriteConfig()
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
			// 因为mustExist必须再重新来查一次
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
			currentChildren, err = list(ListReq{
				// 因为mustExist必须再重新来查一次
				Reload: false,
				Dir:    target,
			})
			if err != nil {
				return nil, OnlyError(err)
			}
			for _, file := range currentChildren {
				if file.Name == pathStr {
					target = file
					exist = true
					break
				}
			}
			if !exist {
				return nil, OnlyMsg(fmt.Sprintf("%s not found", path))
			}
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
		uploaded := pr.uploaded + pr.currentUploaded
		// 相等即已经处理完毕
		if pr.currentSize == pr.currentUploaded {
			pr.uploaded += pr.currentSize
		}
		if pr.uploaded == pr.totalSize {
			pr.finish = true
		}
		internal.LogProgress("uploading", pr.file.Name(), pr.startTime, pr.currentUploaded, uploaded, pr.totalSize, false)
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
	pr.currentSize = endSize - startSize
	pr.currentUploaded = 0
	return startSize, endSize
}

func (pr *ProgressReader) Close() {
	if pr.file != nil {
		pr.file.Close()
	}
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
