package quark

import (
	"fmt"
	"github.com/hefeiyu2025/pan-client/internal"
	"github.com/hefeiyu2025/pan-client/pan"
	"github.com/imroc/req/v3"
	logger "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	CookiePusKey     = "__pus"
	CookiePuusKey    = "__puus"
	HeaderUserAgent  = "User-Agent"
	DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) quark-cloud-drive/2.5.20 Chrome/100.0.4896.160 Electron/18.3.5.4-b478491100 Safari/537.36 Channel/pckk_other_ch"
)

type Quark struct {
	sessionClient *req.Client
	defaultClient *req.Client
	properties    *QuarkProperties
	pan.PropertiesOperate
	pan.CacheOperate
	pan.CommonOperate
	pan.BaseOperate
}

type QuarkProperties struct {
	Pus         string `mapstructure:"pus" json:"pus" yaml:"pus"`
	Puus        string `mapstructure:"puus" json:"puus" yaml:"puus"`
	RefreshTime int64  `mapstructure:"refresh_time" json:"refresh_time" yaml:"refresh_time" default:"0"`
	ChunkSize   int64  `mapstructure:"chunk_size" json:"chunk_size" yaml:"chunk_size" default:"104857600"` // 100M
}

func (cp *QuarkProperties) OnlyImportProperties() {
	// do nothing
}

func (q *Quark) Init() error {
	var properties QuarkProperties
	err := q.ReadConfig(&properties)
	if err != nil {
		return err
	}
	q.properties = &properties
	if properties.Pus == "" || properties.Puus == "" {
		_ = q.WriteConfig(q.properties)
		return fmt.Errorf("please set pus and puus")
	}
	q.sessionClient = req.C().
		SetCommonHeaders(map[string]string{
			HeaderUserAgent: DefaultUserAgent,
			"Accept":        "application/json, text/plain, */*",
			"Referer":       "https://pan.quark.cn",
		}).
		SetCommonQueryParam("pr", "ucpro").
		SetCommonQueryParam("fr", "pc").
		SetCommonCookies(&http.Cookie{Name: CookiePusKey, Value: q.properties.Pus}, &http.Cookie{Name: CookiePuusKey, Value: q.properties.Puus}).
		SetTimeout(30 * time.Minute).SetBaseURL("https://drive.quark.cn/1/clouddrive")
	q.defaultClient = req.C().SetTimeout(30 * time.Minute)
	// 若一小时内更新过，则不重新刷session
	if q.properties.RefreshTime == 0 || q.properties.RefreshTime-time.Now().UnixMilli() > 60*60*1000 {
		_, err = q.config()
		if err != nil {
			return err
		} else {
			err = q.WriteConfig(q.properties)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (q *Quark) Drop() error {
	return pan.OnlyMsg("not support")
}

func (q *Quark) Disk() (*pan.DiskResp, error) {
	return nil, pan.OnlyMsg("not support")
	//storageResp, err := q.userStorage()
	//if err != nil {
	//	return nil, err
	//}
	//return &pan.DiskResp{
	//	Total: storageResp.Data.Total / 1024 / 1024,
	//	Free:  storageResp.Data.Free / 1024 / 1024,
	//	Used:  storageResp.Data.Used / 1024 / 1024,
	//}, nil
}
func (q *Quark) List(req pan.ListReq) ([]*pan.PanObj, error) {
	if req.Dir.Path == "/" && req.Dir.Name == "" {
		req.Dir.Id = "0"
	}
	cacheKey := cacheDirectoryPrefix + req.Dir.Id
	if req.Reload {
		q.Del(cacheKey)
	}
	panObjs, exist, err := q.GetOrDefault(cacheKey, func() (interface{}, error) {
		files, e := q.fileSort(req.Dir.Id)
		if e != nil {
			logger.Error(e)
			return nil, e
		}
		panObjs := make([]*pan.PanObj, 0)
		for _, item := range files {
			fileType := "file"
			if item.FileType == 0 {
				fileType = "dir"
			}
			path := req.Dir.Path + "/" + req.Dir.Name + "/" + item.FileName
			if req.Dir.Id == "0" {
				path = "/" + item.FileName
			}
			panObjs = append(panObjs, &pan.PanObj{
				Id:     item.Fid,
				Name:   item.FileName,
				Path:   path,
				Size:   int64(item.Size),
				Type:   fileType,
				Parent: req.Dir,
			})
		}
		return panObjs, nil
	})
	if err != nil {
		return make([]*pan.PanObj, 0), err
	}
	if exist {
		objs, ok := panObjs.([]*pan.PanObj)
		if ok {
			return objs, nil
		}
	}
	return make([]*pan.PanObj, 0), nil
}
func (q *Quark) ObjRename(req pan.ObjRenameReq) error {
	if req.Obj.Id == "0" || req.Obj.Path == "/" {
		return pan.OnlyMsg("not support rename root path")
	}
	object := req.Obj
	if object.Id == "" {
		path := strings.Trim(req.Obj.Path, "/") + "/" + req.Obj.Name
		obj, err := q.GetPanObj(path, true, q.List)
		if err != nil {
			return err
		}
		object = obj
	}
	err := q.objectRename(object.Id, req.NewName)
	if err != nil {
		return err
	}
	q.Del(cacheDirectoryPrefix + object.Parent.Id)
	return nil
}
func (q *Quark) BatchRename(req pan.BatchRenameReq) error {
	objs, err := q.List(pan.ListReq{
		Reload: true,
		Dir:    req.Path,
	})
	if err != nil {
		return err
	}
	for _, object := range objs {
		if object.Type == "dir" {
			err = q.BatchRename(pan.BatchRenameReq{
				Path: object,
				Func: req.Func,
			})
			if err != nil {
				return err
			}
		}
		newName := req.Func(object)

		if newName != object.Name {
			err = q.ObjRename(pan.ObjRenameReq{
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
func (q *Quark) Mkdir(req pan.MkdirReq) (*pan.PanObj, error) {
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
	obj, err := q.GetPanObj(targetPath, false, q.List)
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
			return nil, pan.OnlyError(err)
		}
		split := strings.Split(rel, "/")
		targetDirId := req.Parent.Id
		tPath := existPath
		for _, s := range split {
			tPath = tPath + "/" + s
			resp, err := q.createDirectory(existPath+"/"+split[0], targetDirId)
			if err != nil {
				return nil, pan.OnlyError(err)
			}
			targetDirId = resp.Data.Fid
		}
		q.Del(cacheDirectoryPrefix + obj.Id)
		return q.Mkdir(req)
	}
}
func (q *Quark) Move(req pan.MovieReq) error {
	targetObj := req.TargetObj
	if targetObj.Type == "file" {
		return pan.OnlyMsg("target is a file")
	}
	// 重新直接创建目标目录
	if targetObj.Id == "" {
		create, err := q.Mkdir(pan.MkdirReq{
			NewPath: strings.Trim(targetObj.Path, "/") + "/" + targetObj.Name,
		})
		if err != nil {
			return err
		}
		targetObj = create
	}
	reloadDirId := make(map[string]any)
	objIds := make([]string, 0)
	for _, item := range req.Items {
		if item.Id != "0" && item.Id != "" {
			objIds = append(objIds, item.Id)
			if item.Type == "dir" {
				reloadDirId[item.Id] = true
			}
		} else if item.Path != "" && item.Path != "/" {
			obj, err := q.GetPanObj(item.Path, true, q.List)
			if err == nil {
				objIds = append(objIds, obj.Id)
				if obj.Type == "dir" {
					reloadDirId[obj.Id] = true
				}
			}
		}
	}
	err := q.objectMove(objIds, req.TargetObj.Id)
	if err != nil {
		return pan.OnlyError(err)
	}
	for key, _ := range reloadDirId {
		q.Del(cacheDirectoryPrefix + key)
	}
	return nil
}
func (q *Quark) Delete(req pan.DeleteReq) error {
	if len(req.Items) == 0 {
		return nil
	}
	reloadDirId := make(map[string]any)
	objIds := make([]string, 0)
	for _, item := range req.Items {
		if item.Id != "0" && item.Id != "" {
			objIds = append(objIds, item.Id)
			if item.Type == "dir" {
				reloadDirId[item.Id] = true
			}
		} else if item.Path != "" && item.Path != "/" {
			obj, err := q.GetPanObj(item.Path, true, q.List)
			if err == nil {
				objIds = append(objIds, obj.Id)
				if obj.Type == "dir" {
					reloadDirId[obj.Id] = true
				}
			}
		}
	}
	if len(objIds) > 0 {
		err := q.objectDelete(objIds)
		if err != nil {
			return err
		}
		for key, _ := range reloadDirId {
			q.Del(cacheDirectoryPrefix + key)
		}
	}

	return nil
}

func (q *Quark) UploadPath(req pan.UploadPathReq) error {
	return q.BaseUploadPath(req, q.UploadFile)
}

func (q *Quark) uploadErrAfter(md5Key string, uploadedSize int64) {
	q.Set(cacheChunkPrefix+md5Key, uploadedSize)
	errorTimes, _, _ := q.GetOrDefault(cacheSessionErrPrefix+md5Key, func() (interface{}, error) {
		return 0, nil
	})
	i := errorTimes.(int)
	if i > 3 {
		q.Del(cacheSessionPrefix + md5Key)
		q.Del(cacheMd5sPrefix + md5Key)
		q.Del(cacheChunkPrefix + md5Key)
		q.Del(cacheSessionErrPrefix + md5Key)
	}
	q.Set(cacheSessionErrPrefix+md5Key, i+1)
}

func (q *Quark) UploadFile(req pan.UploadFileReq) error {
	md5Str, err := internal.GetFileMd5(req.LocalFile)
	if err != nil {
		return err
	}
	sha1Str, err := internal.GetFileSha1(req.LocalFile)
	if err != nil {
		return err
	}

	mimeType := internal.GetMimeType(req.LocalFile)

	file, err := os.Open(req.LocalFile)
	if err != nil {
		return err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	remoteName := stat.Name()
	remotePath := req.RemotePath
	if req.RemoteTransfer != nil {
		remoteName, remotePath = req.RemoteTransfer(remoteName, remotePath)
	}
	remoteAllPath := remotePath + "/" + remoteName
	_, err = q.GetPanObj(remoteAllPath, true, q.List)
	// 没有报错证明文件已经存在
	if err == nil {
		return pan.CodeMsg(CodeObjectExist, remoteAllPath+" is exist")
	}
	dir, err := q.Mkdir(pan.MkdirReq{
		NewPath: remotePath,
	})
	if err != nil {
		return pan.MsgError(remotePath+" create error", err)
	}

	md5Key := internal.Md5HashStr(req.LocalFile + remotePath + dir.Id)
	if !req.Resumable {
		q.Del(cacheSessionPrefix + md5Key)
		q.Del(cacheChunkPrefix + md5Key)
		q.Del(cacheMd5sPrefix + md5Key)
		q.Del(cacheSessionErrPrefix + md5Key)
	}
	var uploadedSize int64 = 0
	if obj, exist := q.Get(cacheChunkPrefix + md5Key); exist {
		uploadedSize = obj.(int64)
	}
	md5s := make([]string, 0)
	if obj, exist := q.Get(cacheChunkPrefix + md5Key); exist {
		md5s = obj.([]string)
	}

	var pre RespDataWithMeta[FileUpPre, FileUpPreMeta]
	data, exist, e := q.GetOrDefault(cacheSessionPrefix+md5Key, func() (interface{}, error) {
		// pre
		resp, err := q.FileUploadPre(FileUpPreReq{
			ParentId: dir.Id,
			FileName: remoteName,
			FileSize: stat.Size(),
			MimeType: mimeType,
		})
		if err != nil {
			return nil, err
		}
		pre = *resp
		return *resp, nil
	})
	if e != nil {
		return e
	}
	if exist {
		pre = data.(RespDataWithMeta[FileUpPre, FileUpPreMeta])
	}

	// hash
	finish, err := q.FileUploadHash(FileUpHashReq{
		Md5:    md5Str,
		Sha1:   sha1Str,
		TaskId: pre.Data.TaskId,
	})
	if err != nil {
		q.uploadErrAfter(md5Key, uploadedSize)
		return err
	}
	if finish.Data.Finish {
		return nil
	}

	// part up
	partSize := int64(pre.Metadata.PartSize)
	total := stat.Size()
	left := total - uploadedSize
	if uploadedSize > 0 {
		// 将文件指针移动到指定的分片位置
		ret, _ := file.Seek(uploadedSize, 0)
		if ret == 0 {
			return fmt.Errorf("seek file failed")
		}
	}
	partNumber := (uploadedSize / partSize) + 1
	pr, err := pan.NewProcessReader(req.LocalFile, partSize, uploadedSize)
	if err != nil {
		return err
	}
	for left > 0 {
		start, end := pr.NextChunk()
		chunkUploadSize := end - start
		left -= chunkUploadSize
		m, err := q.FileUpPart(FileUpPartReq{
			ObjKey:     pre.Data.ObjKey,
			Bucket:     pre.Data.Bucket,
			UploadId:   pre.Data.UploadId,
			AuthInfo:   pre.Data.AuthInfo,
			UploadUrl:  pre.Data.UploadUrl,
			MineType:   mimeType,
			PartNumber: int(partNumber),
			TaskId:     pre.Data.TaskId,
			Reader:     pr,
		})
		if err != nil {
			q.uploadErrAfter(md5Key, uploadedSize)
			return err
		}
		if m == "finish" {
			return nil
		}
		md5s = append(md5s, m)
		if req.Resumable {
			q.Set(cacheChunkPrefix+md5Key, uploadedSize+chunkUploadSize)
			q.Set(cacheMd5sPrefix+md5Key, md5s)
		}
		partNumber++
	}
	err = q.FileUpCommit(FileUpCommitReq{
		ObjKey:    pre.Data.ObjKey,
		Bucket:    pre.Data.Bucket,
		UploadId:  pre.Data.UploadId,
		AuthInfo:  pre.Data.AuthInfo,
		UploadUrl: pre.Data.UploadUrl,
		MineType:  mimeType,
		TaskId:    pre.Data.TaskId,
		Callback:  pre.Data.Callback,
	}, md5s)
	if err != nil {
		return err
	}
	_, err = q.FileUpFinish(FileUpFinishReq{
		ObjKey: pre.Data.ObjKey,
		TaskId: pre.Data.TaskId,
	})
	if err != nil {
		q.uploadErrAfter(md5Key, uploadedSize)
		return err
	}
	if req.Resumable {
		q.Del(cacheSessionPrefix + md5Key)
		q.Del(cacheChunkPrefix + md5Key)
		q.Del(cacheMd5sPrefix + md5Key)
		q.Del(cacheSessionErrPrefix + md5Key)
	}
	logger.Infof("upload success:%s", req.LocalFile)
	// 上传成功则移除文件了
	if req.SuccessDel {
		_ = os.Remove(req.LocalFile)
		logger.Infof("delete success   %s", req.LocalFile)
	}
	return nil
}

func (q *Quark) DownloadPath(req pan.DownloadPathReq) error {
	return q.BaseDownloadPath(req, q.List, q.DownloadFile)
}
func (q *Quark) DownloadFile(req pan.DownloadFileReq) error {
	return q.BaseDownloadFile(req, q.sessionClient, func(req pan.DownloadFileReq) (string, error) {
		resp, err := q.fileDownload(req.RemoteFile.Id)
		if err != nil {
			return "", err
		}
		return resp.Data[0].DownloadUrl, nil
	})
}

func (q *Quark) ShareList() {}
func (q *Quark) NewShare() {

}
func (q *Quark) DeleteShare() {

}

func init() {
	pan.RegisterDriver(pan.Quark, func() pan.Driver {
		return &Quark{
			PropertiesOperate: pan.PropertiesOperate{
				DriverType: pan.Quark,
			},
			CacheOperate:  pan.CacheOperate{DriverType: pan.Quark},
			CommonOperate: pan.CommonOperate{},
		}
	})
}
