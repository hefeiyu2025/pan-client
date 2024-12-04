package cloudreve

import (
	"fmt"
	"github.com/hefeiyu2025/pan-client/client"
	"github.com/hefeiyu2025/pan-client/internal"
	"github.com/imroc/req/v3"
	logger "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	CookieSessionKey = "cloudreve-session"
	HeaderUserAgent  = "User-Agent"
	DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36"
)

type Cloudreve struct {
	sessionClient *req.Client
	defaultClient *req.Client
	properties    *CloudreveProperties
	client.PropertiesOperate
	client.CacheOperate
}

type CloudreveProperties struct {
	CloudreveUrl     string `mapstructure:"url" json:"url" yaml:"url"`
	CloudreveSession string `mapstructure:"session" json:"session" yaml:"session"`
	RefreshTime      int64  `mapstructure:"refresh_time" json:"refresh_time" yaml:"refresh_time" defualt:"0"`
}

func (cp *CloudreveProperties) OnlyImportProperties() {
	// do nothing
}

func (c *Cloudreve) Init() error {
	var properties CloudreveProperties
	err := c.ReadConfig(&properties)
	if err != nil {
		return err
	}
	c.properties = &properties
	if properties.CloudreveUrl == "" || properties.CloudreveSession == "" {
		_ = c.WriteConfig(c.properties)
		return fmt.Errorf("please set cloudreve url and session")
	}
	c.sessionClient = req.C().SetCommonHeader(HeaderUserAgent, DefaultUserAgent).
		SetCommonHeader("Accept", "application/json, text/plain, */*").
		SetTimeout(30 * time.Minute).SetBaseURL(c.properties.CloudreveUrl + "/api/v3").
		SetCommonCookies(&http.Cookie{Name: CookieSessionKey, Value: c.properties.CloudreveSession})
	c.defaultClient = req.C().SetCommonHeader(HeaderUserAgent, DefaultUserAgent).SetTimeout(2 * time.Hour)
	c.defaultClient.GetTransport().
		WrapRoundTripFunc(func(rt http.RoundTripper) req.HttpRoundTripFunc {
			return func(req *http.Request) (resp *http.Response, err error) {
				// 由于内容长度部分是由后台计算的，所以这里需要手动设置,http默认会过滤掉header.reqWriteExcludeHeader
				if req.ContentLength <= 0 {
					if req.Header.Get("Content-Length") != "" {
						req.ContentLength, _ = strconv.ParseInt(req.Header.Get("Content-Length"), 10, 64)
					}
				}
				return rt.RoundTrip(req)
			}
		})
	// 若一小时内更新过，则不重新刷session
	if c.properties.RefreshTime == 0 || c.properties.RefreshTime-time.Now().UnixMilli() > 60*60*1000 {
		_, err = c.config()
		if err != nil {
			return err
		} else {
			err = c.WriteConfig(c.properties)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Cloudreve) Drop() error {
	return OnlyMsg("not support")
}

func (c *Cloudreve) Disk() (*client.DiskResp, error) {
	storageResp, err := c.userStorage()
	if err != nil {
		return nil, err
	}
	return &client.DiskResp{
		Total: storageResp.Data.Total / 1024 / 1024,
		Free:  storageResp.Data.Free / 1024 / 1024,
		Used:  storageResp.Data.Used / 1024 / 1024,
	}, nil
}
func (c *Cloudreve) List(req client.ListReq) ([]*client.PanObj, error) {
	if req.Dir.Path == "/" {
		req.Dir.Id = "0"
	}
	cacheKey := "directory_" + req.Dir.Id
	if req.Reload {
		c.Del(cacheKey)
	}
	panObjs, exist, err := c.GetOrDefault(cacheKey, func() (interface{}, error) {
		directory, e := c.listDirectory(req.Dir.Path)
		if e != nil {
			logger.Error(e)
			return nil, e
		}
		panObjs := make([]*client.PanObj, 0)
		for _, item := range directory.Data.Objects {
			panObjs = append(panObjs, &client.PanObj{
				Id:     item.ID,
				Name:   item.Name,
				Path:   item.Path,
				Size:   item.Size,
				Type:   item.Type,
				Parent: req.Dir,
			})
		}
		c.Set("policy", directory.Data.Policy)
		return panObjs, nil
	})
	if err != nil {
		return make([]*client.PanObj, 0), err
	}
	if exist {
		objs, ok := panObjs.([]*client.PanObj)
		if ok {
			return objs, nil
		}
	}
	return make([]*client.PanObj, 0), nil
}
func (c *Cloudreve) Rename() {

}
func (c *Cloudreve) Mkdir()  {}
func (c *Cloudreve) Move()   {}
func (c *Cloudreve) Delete() {}
func (c *Cloudreve) UploadPath(req client.OneStepUploadPathReq) error { // 遍历目录
	err := filepath.Walk(req.LocalPath, func(path string, info os.FileInfo, err error) error {
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
			relPath, _ := filepath.Rel(req.LocalPath, path)
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
				err = c.UploadFile(client.OneStepUploadFileReq{
					LocalFile:      path,
					RemotePath:     strings.TrimRight(req.RemotePath, "/") + "/" + relPath,
					PolicyId:       req.PolicyId,
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
						fmt.Println("upload err", err)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
func (c *Cloudreve) UploadFile(req client.OneStepUploadFileReq) error {
	file, err := os.Open(req.LocalFile)
	if err != nil {
		return err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	remotePath := strings.TrimLeft(req.RemotePath, "/")
	remoteName := stat.Name()
	md5Key := internal.Md5HashStr(req.LocalFile + remotePath + req.PolicyId)
	var session UploadCredential
	if req.Resumable {
		data, exist, err := c.GetOrDefault("session_"+md5Key, func() (interface{}, error) {
			resp, e := c.fileUploadGetUploadSession(CreateUploadSessionReq{
				Path:         "/" + remotePath,
				Size:         uint64(stat.Size()),
				Name:         remoteName,
				PolicyID:     req.PolicyId,
				LastModified: stat.ModTime().UnixMilli(),
			})
			if e != nil {
				return nil, e
			}
			return resp.Data, nil
		})
		if err != nil {
			return err
		}
		if exist {
			session = data.(UploadCredential)
		}

	}
	if req.RemoteTransfer != nil {
		remotePath, remoteName = req.RemoteTransfer(remotePath, remoteName)
	}

	uploadedSize := 0
	if req.Resumable {
		//cacheErr := GetCache("chunk_"+md5Key, &uploadedSize)
		//if cacheErr != nil {
		//	fmt.Println("cache err:", cacheErr)
		//}
	}
	_, err = c.OneDriveUpload(OneDriveUploadReq{
		UploadUrl:    session.UploadURLs[0],
		LocalFile:    file,
		UploadedSize: int64(uploadedSize),
		ChunkSize:    int64(session.ChunkSize),
	})
	if err != nil {
		//dealError(req.Resumable, md5Key, session.SessionID, uploaded, c)
		return err
	}

	_, err = c.oneDriveCallback(session.SessionID)
	if err != nil {
		//dealError(req.Resumable, md5Key, session.SessionID, uploaded, c)
		return err
	}
	if req.Resumable {
		//_ = DelCache("session_" + md5Key)
		//_ = DelCache("chunk_" + md5Key)
	}
	// 上传成功则移除文件了
	if req.SuccessDel {
		_ = os.Remove(req.LocalFile)
		fmt.Println("uploaded success and delete", req.LocalFile)
	}
	return nil
}
func (c *Cloudreve) DownloadPath(req client.OneStepDownloadPathReq) error { return nil }
func (c *Cloudreve) DownloadFile(req client.OneStepDownloadFileReq) error { return nil }

func (c *Cloudreve) ShareList() {}
func (c *Cloudreve) NewShare() {

}
func (c *Cloudreve) DeleteShare() {

}

func init() {
	client.RegisterDriver(client.Cloudreve, func() client.Driver {
		return &Cloudreve{
			PropertiesOperate: client.PropertiesOperate{
				DriverType: client.Cloudreve,
			},
			CacheOperate: client.CacheOperate{DriverType: client.Cloudreve},
		}
	})
}
