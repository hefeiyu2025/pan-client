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
	client.CommonOperate
	client.BaseOperate
}

type CloudreveProperties struct {
	CloudreveUrl     string `mapstructure:"url" json:"url" yaml:"url"`
	CloudreveSession string `mapstructure:"session" json:"session" yaml:"session"`
	RefreshTime      int64  `mapstructure:"refresh_time" json:"refresh_time" yaml:"refresh_time" default:"0"`
	ChunkSize        int64  `mapstructure:"chunk_size" json:"chunk_size" yaml:"chunk_size" default:"104857600"` // 100M
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
	return client.OnlyMsg("not support")
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
	if req.Dir.Path == "/" && req.Dir.Name == "" {
		req.Dir.Id = "0"
	}
	cacheKey := cacheDirectoryPrefix + req.Dir.Id
	if req.Reload {
		c.Del(cacheKey)
	}
	panObjs, exist, err := c.GetOrDefault(cacheKey, func() (interface{}, error) {
		directory, e := c.listDirectory(strings.TrimRight(req.Dir.Path, "/") + "/" + req.Dir.Name)
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
		c.Set(cachePolicy, directory.Data.Policy)
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
func (c *Cloudreve) ObjRename(req client.ObjRenameReq) error {
	if req.Obj.Id == "0" || req.Obj.Path == "/" {
		return client.OnlyMsg("not support rename root path")
	}
	object := req.Obj
	if object.Id == "" {
		path := strings.Trim(req.Obj.Path, "/") + "/" + req.Obj.Name
		obj, err := c.GetPanObj(path, true, c.List)
		if err != nil {
			return err
		}
		object = obj
	}
	item := Item{}
	if object.Type == "dir" {
		item.Dirs = []string{object.Id}
	} else {
		item.Items = []string{object.Id}
	}
	_, err := c.objectRename(ItemRenameReq{Src: item,
		NewName: req.NewName})
	if err != nil {
		return err
	}
	c.Del(cacheDirectoryPrefix + object.Parent.Id)
	return nil
}
func (c *Cloudreve) BatchRename(req client.BatchRenameReq) error {
	objs, err := c.List(client.ListReq{
		Reload: true,
		Dir:    req.Path,
	})
	if err != nil {
		return err
	}
	for _, object := range objs {
		if object.Type == "dir" {
			err = c.BatchRename(client.BatchRenameReq{
				Path: object,
				Func: req.Func,
			})
			if err != nil {
				return err
			}
		}
		newName := req.Func(object)

		if newName != object.Name {
			err = c.ObjRename(client.ObjRenameReq{
				Obj:     object,
				NewName: newName,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}
func (c *Cloudreve) Mkdir(req client.MkdirReq) (*client.PanObj, error) {
	if req.NewPath == "" || filepath.Ext(req.NewPath) != "" {
		// 不处理，直接返回
		return nil, nil
	}
	targetPath := "/"
	if req.Parent != nil {
		targetPath = req.Parent.Path + "/" + strings.Trim(req.NewPath, "/")
		if req.Parent.Id == "0" || req.Parent.Path == "/" {
			targetPath = "/" + strings.Trim(req.NewPath, "/")
		}
	}
	obj, err := c.GetPanObj(targetPath, false, c.List)
	if err != nil {
		return nil, err
	}
	existPath := obj.Path + "/" + obj.Name
	if obj.Id == "0" || obj.Path == "/" {
		existPath = "/" + obj.Name
	}
	if existPath == targetPath {
		return obj, nil
	} else {
		rel, err := filepath.Rel(existPath, targetPath)
		rel = strings.Replace(rel, "\\", "/", -1)
		if err != nil {
			return nil, client.OnlyError(err)
		}
		split := strings.Split(rel, "/")

		_, err = c.createDirectory(existPath + "/" + split[0])
		if err != nil {
			return nil, client.OnlyError(err)
		}
		c.Del(cacheDirectoryPrefix + obj.Id)
		return c.Mkdir(req)
	}
}
func (c *Cloudreve) Move(req client.MovieReq) error {
	sameSrc := make(map[string][]*client.PanObj)
	for _, item := range req.Items {
		objs, ok := sameSrc[item.Path]
		if ok {
			objs = append(objs, item)
			sameSrc[item.Path] = objs
		} else {
			sameSrc[item.Path] = []*client.PanObj{item}
		}
	}
	targetObj := req.TargetObj
	if targetObj.Type == "file" {
		return client.OnlyMsg("target is a file")
	}
	// 重新直接创建目标目录
	if targetObj.Id == "" {
		create, err := c.Mkdir(client.MkdirReq{
			NewPath: strings.Trim(targetObj.Path, "/") + "/" + targetObj.Name,
		})
		if err != nil {
			return err
		}
		targetObj = create
	}
	for src, items := range sameSrc {
		reloadDirId := make(map[string]any)
		itemIds := make([]string, 0)
		dirIds := make([]string, 0)
		for _, item := range items {
			if item.Id != "0" && item.Id != "" {
				if item.Type == "dir" {
					dirIds = append(dirIds, item.Id)
					reloadDirId[item.Id] = true
				} else {
					itemIds = append(itemIds, item.Id)
				}
			} else if item.Path != "" && item.Path != "/" {
				obj, err := c.GetPanObj(strings.Trim(item.Path, "/")+"/"+item.Name, true, c.List)
				if err == nil {
					if obj.Type == "dir" {
						dirIds = append(dirIds, obj.Id)
						reloadDirId[obj.Id] = true
					} else {
						itemIds = append(itemIds, obj.Id)
						reloadDirId[obj.Parent.Id] = true
					}
				}
			}
		}
		_, err := c.objectMove(ItemMoveReq{
			SrcDir: src,
			Dst:    strings.Trim(targetObj.Path, "/") + "/" + targetObj.Name,
			Src: Item{
				Items: itemIds,
				Dirs:  dirIds,
			},
		})
		if err != nil {
			return client.OnlyError(err)
		}
		reloadDirId[targetObj.Id] = true
		for key, _ := range reloadDirId {
			c.Del(cacheDirectoryPrefix + key)
		}
	}
	return nil
}
func (c *Cloudreve) Delete(req client.DeleteReq) error {
	if len(req.Items) == 0 {
		return nil
	}
	reloadDirId := make(map[string]any)
	itemIds := make([]string, 0)
	dirIds := make([]string, 0)
	for _, item := range req.Items {
		if item.Id != "0" && item.Id != "" {
			if item.Type == "dir" {
				dirIds = append(dirIds, item.Id)
				reloadDirId[item.Id] = true
			} else {
				itemIds = append(itemIds, item.Id)
			}
		} else if item.Path != "" && item.Path != "/" {
			obj, err := c.GetPanObj(item.Path, true, c.List)
			if err == nil {
				if obj.Type == "dir" {
					dirIds = append(dirIds, obj.Id)
					reloadDirId[obj.Id] = true
				} else {
					itemIds = append(itemIds, obj.Id)
					reloadDirId[obj.Parent.Id] = true
				}
			}
		}
	}
	if len(itemIds) > 0 || len(dirIds) > 0 {
		_, err := c.objectDelete(ItemReq{
			Item: Item{
				Items: itemIds,
				Dirs:  dirIds,
			},
			Force:      true,
			UnlinkOnly: false,
		})
		if err != nil {
			return err
		}
		for key, _ := range reloadDirId {
			c.Del(cacheDirectoryPrefix + key)
		}
	}

	return nil
}

func (c *Cloudreve) UploadPath(req client.OneStepUploadPathReq) error {
	return c.BaseUploadPath(req, c.UploadFile)
}

func (c *Cloudreve) uploadErrAfter(md5Key string, uploadedSize int64, session UploadCredential) {
	c.Set(cacheChunkPrefix+md5Key, uploadedSize)
	errorTimes, _, _ := c.GetOrDefault(cacheSessionErrPrefix+md5Key, func() (interface{}, error) {
		return 0, nil
	})
	i := errorTimes.(int)
	if i > 3 {
		if session.SessionID != "" {
			_, _ = c.fileUploadDeleteUploadSession(session.SessionID)
		} else {
			_, _ = c.fileUploadDeleteAllUploadSession()
		}
		c.Del(cacheSessionPrefix + md5Key)
		c.Del(cacheChunkPrefix + md5Key)
		c.Del(cacheSessionErrPrefix + md5Key)
	}
	c.Set(cacheSessionErrPrefix+md5Key, i+1)
}

func (c *Cloudreve) UploadFile(req client.OneStepUploadFileReq) error {
	stat, err := os.Stat(req.LocalFile)
	if err != nil {
		return err
	}
	remotePath := strings.Trim(req.RemotePath, "/")
	remoteName := stat.Name()
	if req.RemoteTransfer != nil {
		remotePath, remoteName = req.RemoteTransfer(remotePath, remoteName)
	}
	remoteAllPath := remotePath + "/" + remoteName
	_, err = c.GetPanObj(remoteAllPath, true, c.List)
	// 没有报错证明文件已经存在
	if err == nil {
		return client.CodeMsg(CodeObjectExist, remoteAllPath+" is exist")
	}
	_, err = c.Mkdir(client.MkdirReq{
		NewPath: remotePath,
	})
	if err != nil {
		return client.MsgError(remotePath+" create error", err)
	}
	md5Key := internal.Md5HashStr(remoteAllPath)
	if !req.Resumable {
		c.Del(cacheSessionPrefix + md5Key)
		c.Del(cacheChunkPrefix + md5Key)
		c.Del(cacheSessionErrPrefix + md5Key)
	}
	var uploadedSize int64 = 0
	if obj, exist := c.Get(cacheChunkPrefix + md5Key); exist {
		uploadedSize = obj.(int64)
	}
	var session UploadCredential
	data, exist, e := c.GetOrDefault(cacheSessionPrefix+md5Key, func() (interface{}, error) {
		policy, exist := c.Get(cachePolicy)
		if !exist {
			return nil, client.OnlyMsg(cachePolicy + " is not exist")
		}
		summary := policy.(*PolicySummary)
		resp, e := c.fileUploadGetUploadSession(CreateUploadSessionReq{
			Path:         "/" + remotePath,
			Size:         uint64(stat.Size()),
			Name:         remoteName,
			PolicyID:     summary.ID,
			LastModified: stat.ModTime().UnixMilli(),
		})
		if e != nil {
			if e.GetCode() == CodeConflictUploadOngoing {
				// 要是存在重复的文件，直接删掉别的seesion再上传
				_, _ = c.fileUploadDeleteAllUploadSession()
				sResp, secE := c.fileUploadGetUploadSession(CreateUploadSessionReq{
					Path:         "/" + remotePath,
					Size:         uint64(stat.Size()),
					Name:         remoteName,
					PolicyID:     summary.ID,
					LastModified: stat.ModTime().UnixMilli(),
				})
				if secE != nil {
					return nil, secE
				}
				return sResp.Data, nil
			}
			return nil, e
		}
		return resp.Data, nil
	})
	if e != nil {
		return e
	}
	if exist {
		session = data.(UploadCredential)
	}

	uploadedSize, err = c.oneDriveUpload(OneDriveUploadReq{
		UploadUrl:    session.UploadURLs[0],
		LocalFile:    req.LocalFile,
		UploadedSize: uploadedSize,
		ChunkSize:    min(int64(session.ChunkSize), c.properties.ChunkSize),
	})
	if err != nil {
		c.uploadErrAfter(md5Key, uploadedSize, session)
		return err
	}

	_, err = c.oneDriveCallback(session.SessionID)
	if err != nil {
		c.uploadErrAfter(md5Key, uploadedSize, session)
		return err
	}
	if req.Resumable {
		c.Del(cacheSessionPrefix + md5Key)
		c.Del(cacheChunkPrefix + md5Key)
		c.Del(cacheSessionErrPrefix + md5Key)
	}
	// 上传成功则移除文件了
	if req.SuccessDel {
		_ = os.Remove(req.LocalFile)
		logger.Println("uploaded success and delete", req.LocalFile)
	}
	return nil
}

func (c *Cloudreve) DownloadPath(req client.OneStepDownloadPathReq) error {
	return c.BaseDownloadPath(req, c.List, c.DownloadFile)
}
func (c *Cloudreve) DownloadFile(req client.OneStepDownloadFileReq) error {
	return c.BaseDownloadFile(req, c.defaultClient, func(req client.OneStepDownloadFileReq) (string, error) {
		resp, err := c.fileCreateDownloadSession(req.RemoteFile.Id)
		if err != nil {
			return "", err
		}
		return resp.Data, nil
	})
}

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
			CacheOperate:  client.CacheOperate{DriverType: client.Cloudreve},
			CommonOperate: client.CommonOperate{},
		}
	})
}
