package quark

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/hefeiyu2025/pan-client/internal"
	"github.com/hefeiyu2025/pan-client/pan"
	"github.com/imroc/req/v3"
	logger "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Quark struct {
	sessionClient *req.Client
	defaultClient *req.Client
	pan.PropertiesOperate[*QuarkProperties]
	pan.CacheOperate
	pan.CommonOperate
	pan.BaseOperate
}

type QuarkProperties struct {
	Id          string `mapstructure:"id" json:"id" yaml:"id"`
	Pus         string `mapstructure:"pus" json:"pus" yaml:"pus"`
	Puus        string `mapstructure:"puus" json:"puus" yaml:"puus"`
	RefreshTime int64  `mapstructure:"refresh_time" json:"refresh_time" yaml:"refresh_time" default:"0"`
	ChunkSize   int64  `mapstructure:"chunk_size" json:"chunk_size" yaml:"chunk_size" default:"314572800"` // 300M
}

func (cp *QuarkProperties) OnlyImportProperties() {
	// do nothing
}

func (cp *QuarkProperties) GetId() string {
	if cp.Id == "" {
		cp.Id = uuid.NewString()
	}
	return cp.Id
}

func (cp *QuarkProperties) GetDriverType() pan.DriverType {
	return pan.Quark
}

func (q *Quark) Init() (string, error) {
	err := q.ReadConfig()
	if err != nil {
		return "", err
	}
	driverId := q.GetId()
	if q.Properties.Pus == "" || q.Properties.Puus == "" {
		_ = q.WriteConfig()
		return driverId, fmt.Errorf("please set pus and puus")
	}
	q.sessionClient = req.C().
		SetCommonHeaders(map[string]string{
			HeaderUserAgent: DefaultUserAgent,
			"Accept":        "application/json, text/plain, */*",
			"Referer":       "https://pan.quark.cn",
		}).
		SetCommonQueryParam("pr", "ucpro").
		SetCommonQueryParam("fr", "pc").
		SetCommonCookies(&http.Cookie{Name: CookiePusKey, Value: q.Properties.Pus}, &http.Cookie{Name: CookiePuusKey, Value: q.Properties.Puus}).
		SetTimeout(30 * time.Minute).SetBaseURL("https://drive.quark.cn/1/clouddrive")
	q.defaultClient = req.C().SetTimeout(30 * time.Minute)
	// 若一小时内更新过，则不重新刷session
	if q.Properties.RefreshTime == 0 || time.Now().UnixMilli()-q.Properties.RefreshTime > 60*60*1000 {
		_, err = q.config()
		if err != nil {
			return driverId, err
		} else {
			err = q.WriteConfig()
			if err != nil {
				return driverId, err
			}
		}
	}
	return driverId, nil
}

func (q *Quark) InitByCustom(id string, read pan.ConfigRW, write pan.ConfigRW) (string, error) {
	q.Properties = &QuarkProperties{Id: id}
	q.PropertiesOperate.Write = write
	q.PropertiesOperate.Read = read
	return q.Init()
}

func (q *Quark) Drop() error {
	return pan.OnlyMsg("not support")
}

func (q *Quark) Disk() (*pan.DiskResp, error) {
	memberResp, err := q.member()
	if err != nil {
		return nil, err
	}
	return &pan.DiskResp{
		Total: memberResp.Data.TotalCapacity / 1024 / 1024,
		Free:  (memberResp.Data.TotalCapacity - memberResp.Data.UseCapacity) / 1024 / 1024,
		Used:  memberResp.Data.UseCapacity / 1024 / 1024,
	}, nil
}
func (q *Quark) List(req pan.ListReq) ([]*pan.PanObj, error) {
	queryDir := req.Dir
	if queryDir.Path == "/" && queryDir.Name == "" {
		queryDir.Id = "0"
	}
	if queryDir.Id == "" {
		obj, err := q.GetPanObj(strings.TrimRight(queryDir.Path, "/")+"/"+queryDir.Name, true, q.List)
		if err != nil {
			return nil, err
		}
		queryDir = obj
	}
	cacheKey := cacheDirectoryPrefix + queryDir.Id
	if req.Reload {
		q.Del(cacheKey)
	}
	panObjs, exist, err := q.GetOrDefault(cacheKey, func() (interface{}, error) {
		files, e := q.fileSort(queryDir.Id)
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
			path := strings.TrimRight(queryDir.Path, "/") + "/" + queryDir.Name
			if queryDir.Id == "0" {
				path = "/"
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
	if req.Obj.Id == "0" || (req.Obj.Path == "/" && req.Obj.Name == "") {
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
	if req.NewPath == "" {
		// 不处理，直接返回
		return &pan.PanObj{
			Id:   "0",
			Name: "",
			Path: "/",
			Size: 0,
			Type: "dir",
		}, nil
	}
	if filepath.Ext(req.NewPath) != "" {
		return nil, pan.OnlyMsg("please set a dir")
	}
	targetPath := "/" + strings.Trim(req.NewPath, "/")
	if req.Parent != nil && (req.Parent.Id == "0" || req.Parent.Path == "/") {
		targetPath = req.Parent.Path + "/" + strings.Trim(req.NewPath, "/")
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
		targetDirId := obj.Id
		for _, s := range split {
			resp, err := q.createDirectory(s, targetDirId)
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
	err := q.objectMove(objIds, targetObj.Id)
	if err != nil {
		return pan.OnlyError(err)
	}
	for key := range reloadDirId {
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
			} else {
				if item.Parent.Id != "" {
					reloadDirId[item.Parent.Id] = true
				}
			}
		} else if item.Path != "" && item.Path != "/" {
			obj, err := q.GetPanObj(item.Path, true, q.List)
			if err == nil {
				objIds = append(objIds, obj.Id)
				if obj.Type == "dir" {
					reloadDirId[obj.Id] = true
				} else {
					reloadDirId[item.Parent.Id] = true
				}
			}
		}
	}
	if len(objIds) > 0 {
		err := q.objectDelete(objIds)
		if err != nil {
			return err
		}
		for key := range reloadDirId {
			q.Del(cacheDirectoryPrefix + key)
		}
	}

	return nil
}

func (q *Quark) UploadPath(req pan.UploadPathReq) error {
	return q.BaseUploadPath(req, q.UploadFile)
}

func (q *Quark) UploadFile(req pan.UploadFileReq) error {
	if req.Resumable {
		logger.Warn("quark is not support resumeable")
	}
	stat, err := os.Stat(req.LocalFile)
	if err != nil {
		return err
	}
	remoteName := stat.Name()
	remotePath := strings.TrimRight(req.RemotePath, "/")
	if req.RemotePathTransfer != nil {
		remotePath = req.RemotePathTransfer(remotePath)
	}
	if req.RemoteNameTransfer != nil {
		remoteName = req.RemoteNameTransfer(remoteName)
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

	md5Str, err := internal.GetFileMd5(req.LocalFile)
	if err != nil {
		return err
	}
	sha1Str, err := internal.GetFileSha1(req.LocalFile)
	if err != nil {
		return err
	}

	mimeType := internal.GetMimeType(req.LocalFile)

	pre, err := q.FileUploadPre(FileUpPreReq{
		ParentId: dir.Id,
		FileName: remoteName,
		FileSize: stat.Size(),
		MimeType: mimeType,
	})
	if err != nil {
		return err
	}

	// hash
	finish, err := q.FileUploadHash(FileUpHashReq{
		Md5:    md5Str,
		Sha1:   sha1Str,
		TaskId: pre.Data.TaskId,
	})
	if err != nil {
		return err
	}
	if finish.Data.Finish {
		logger.Infof("upload fast success %s", req.LocalFile)
		// 上传成功则移除文件了
		if req.SuccessDel {
			err = os.Remove(req.LocalFile)
			if err != nil {
				logger.Errorf("delete fail %s,%v", req.LocalFile, err)
			} else {
				logger.Infof("delete success %s", req.LocalFile)
			}
		}
		return nil
	}

	if req.OnlyFast {
		logger.Infof("upload fast error %s", req.LocalFile)
		return pan.OnlyMsg("only support fast error:" + req.LocalFile)
	}

	// part up
	partSize := min(int64(pre.Metadata.PartSize), q.Properties.ChunkSize)
	total := stat.Size()
	left := total
	partNumber := 1
	pr, err := pan.NewProcessReader(req.LocalFile, partSize, 0)
	if err != nil {
		return err
	}
	md5s := make([]string, 0)
	for left > 0 {
		start, end := pr.NextChunk()
		chunkUploadSize := end - start
		left -= chunkUploadSize
		m, e := q.FileUpPart(FileUpPartReq{
			ObjKey:     pre.Data.ObjKey,
			Bucket:     pre.Data.Bucket,
			UploadId:   pre.Data.UploadId,
			AuthInfo:   pre.Data.AuthInfo,
			UploadUrl:  pre.Data.UploadUrl,
			MineType:   mimeType,
			PartNumber: partNumber,
			TaskId:     pre.Data.TaskId,
			Reader:     pr,
		})
		if e != nil {
			return e
		}
		if m == "finish" {
			logger.Infof("upload success:%s", req.LocalFile)
			// 上传成功则移除文件了
			if req.SuccessDel {
				err = os.Remove(req.LocalFile)
				if err != nil {
					logger.Errorf("delete fail %s,%v", req.LocalFile, err)
				} else {
					logger.Infof("delete success %s", req.LocalFile)
				}
			}
			return nil
		}
		md5s = append(md5s, m)
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
		return err
	}
	logger.Infof("upload success %s", req.LocalFile)
	// 上传成功则移除文件了
	if req.SuccessDel {
		err = os.Remove(req.LocalFile)
		if err != nil {
			logger.Errorf("delete fail %s,%v", req.LocalFile, err)
		} else {
			logger.Infof("delete success %s", req.LocalFile)
		}
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

func (q *Quark) OfflineDownload(req pan.OfflineDownloadReq) (*pan.Task, error) {
	return nil, pan.OnlyMsg("offline download not support")
}

func (q *Quark) TaskList(req pan.TaskListReq) ([]*pan.Task, error) {
	return nil, pan.OnlyMsg("task list not support")
}

func (q *Quark) ShareList(req pan.ShareListReq) ([]*pan.ShareData, error) {
	needFilter := len(req.ShareIds) > 0
	details, err := q.shareList()
	if err != nil {
		return nil, err
	}
	result := make([]*pan.ShareData, 0)
	for _, share := range details {
		if needFilter {
			exist := false
			for _, shareId := range req.ShareIds {
				if shareId == share.ShareId {
					exist = true
					break
				}
			}
			if !exist {
				continue
			}
		}
		result = append(result, &pan.ShareData{
			ShareId:  share.ShareId,
			ShareUrl: share.ShareUrl,
			PassCode: share.Passcode,
			Title:    share.Title,
		})
	}
	return result, nil
}
func (q *Quark) NewShare(req pan.NewShareReq) (*pan.ShareData, error) {
	urlType := 1
	if req.NeedPassCode {
		urlType = 2
	}
	shareId, err := q.share(ShareReq{
		FidList:     req.Fids,
		Title:       req.Title,
		UrlType:     urlType,
		ExpiredType: req.ExpiredType,
	})
	if err != nil {
		return nil, err
	}
	resp, err := q.sharePassword(shareId)
	if err != nil {
		return nil, err
	}
	return &pan.ShareData{
		ShareId:  shareId,
		ShareUrl: resp.Data.ShareUrl,
		PassCode: resp.Data.Passcode,
		Title:    resp.Data.Title,
	}, nil
}
func (q *Quark) DeleteShare(req pan.DelShareReq) error {
	_, err := q.shareDelete(req.ShareIds)
	return err
}

func (q *Quark) ShareRestore(req pan.ShareRestoreReq) error {
	if req.ShareUrl == "" {
		return pan.OnlyMsg("share url must not null")
	}
	// 解析URL
	parsedURL, err := url.Parse(req.ShareUrl)
	if err != nil {
		return err
	}
	pwdId := strings.TrimLeft(parsedURL.Path, "/s/")
	targetDir, err := q.Mkdir(pan.MkdirReq{
		NewPath: req.TargetDir,
	})
	if err != nil {
		return err
	}
	token, err := q.shareToken(ShareTokenReq{
		PwdId:    pwdId,
		Passcode: req.PassCode,
	})
	if err != nil {
		return err
	}
	stoken := token.Data.Stoken
	detail, err := q.shareDetail(ShareDetailReq{
		PwdId:  pwdId,
		Stoken: stoken,
	})
	if err != nil {
		return err
	}
	fidList := make([]string, 0)
	fidTokenList := make([]string, 0)
	for _, file := range detail.List {
		fidList = append(fidList, file.Fid)
		fidTokenList = append(fidTokenList, file.ShareFidToken)
	}
	err = q.shareRestore(RestoreReq{
		FidList:      fidList,
		FidTokenList: fidTokenList,
		ToPdirFid:    targetDir.Id,
		PwdId:        pwdId,
		Stoken:       stoken,
		PdirFid:      targetDir.Id,
		Scene:        "link",
	})
	return err
}

func (q *Quark) DirectLink(req pan.DirectLinkReq) ([]*pan.DirectLink, error) {
	return nil, pan.OnlyMsg("direct link not support")
}

func init() {
	pan.RegisterDriver(pan.Quark, func() pan.Driver {
		return &Quark{
			PropertiesOperate: pan.PropertiesOperate[*QuarkProperties]{
				DriverType: pan.Quark,
			},
			CacheOperate:  pan.CacheOperate{DriverType: pan.Quark},
			CommonOperate: pan.CommonOperate{},
		}
	})
}
