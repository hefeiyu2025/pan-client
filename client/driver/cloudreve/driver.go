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
	targetPath := req.Parent.Path + "/" + strings.Trim(req.NewPath, "/")
	if req.Parent.Id == "0" || req.Parent.Path == "/" {
		targetPath = "/" + strings.Trim(req.NewPath, "/")
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
			Parent: &client.PanObj{
				Id:   "0",
				Name: "",
				Path: "/",
			},
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
			CacheOperate:  client.CacheOperate{DriverType: client.Cloudreve},
			CommonOperate: client.CommonOperate{},
		}
	})
}
