package thunder_browser

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/hefeiyu2025/pan-client/internal"
	"github.com/hefeiyu2025/pan-client/pan"
	"github.com/imroc/req/v3"
	logger "github.com/sirupsen/logrus"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type ThunderBrowser struct {
	sessionClient  *req.Client
	downloadClient *req.Client
	properties     *ThunderBrowserProperties
	pan.PropertiesOperate
	pan.CacheOperate
	pan.CommonOperate
	pan.BaseOperate
}

type ThunderBrowserProperties struct {
	// 登录方式1
	Username string `mapstructure:"username" json:"username" yaml:"username"`
	Password string `mapstructure:"password" json:"password" yaml:"password"`
	// 登录方式2
	RefreshToken string `mapstructure:"refresh_token" json:"refresh_token" yaml:"refresh_token"`

	// 验证码
	CaptchaToken string `mapstructure:"captcha_token" json:"captcha_token" yaml:"captcha_token"`

	DeviceID string `mapstructure:"device_id" json:"device_id" yaml:"device_id"`

	ExpiresIn int64 `mapstructure:"expires_in" json:"expires_in" yaml:"expires_in"`

	TokenType   string `mapstructure:"token_type" json:"token_type" yaml:"token_type"`
	AccessToken string `mapstructure:"access_token" json:"access_token" yaml:"access_token"`

	Sub    string `mapstructure:"sub" json:"sub" yaml:"sub"`
	UserID string `mapstructure:"user_id" json:"user_id" yaml:"user_id"`
}

func (cp *ThunderBrowserProperties) OnlyImportProperties() {
	// do nothing
}

func (tb *ThunderBrowser) Init() error {

	var properties ThunderBrowserProperties
	err := tb.ReadConfig(&properties)
	if err != nil {
		return err
	}
	tb.properties = &properties
	if (properties.Username == "" || properties.Password == "") && properties.RefreshToken == "" {
		_ = tb.WriteConfig(tb.properties)
		return fmt.Errorf("please set login info ")
	}
	tb.properties.DeviceID = internal.Md5HashStr(tb.properties.Username + tb.properties.Password)
	commonHeaderMap := map[string]string{
		HeaderUserAgent:    BuildCustomUserAgent(PackageName, SdkVersion, ClientVersion),
		"accept":           "application/json;charset=UTF-8",
		"x-device-id":      tb.properties.DeviceID,
		"x-client-id":      ClientID,
		"x-client-version": ClientVersion,
	}
	tb.sessionClient = req.C().SetCommonHeaders(commonHeaderMap)

	_, err = tb.userMe()
	// 若能拿到用户信息，证明已经登录
	if err != nil {

		// refreshToken不为空，则先用token登录
		if tb.properties.RefreshToken != "" {
			tb.properties.DeviceID = internal.Md5HashStr(tb.properties.RefreshToken)
			_, loginErr := tb.refreshToken(tb.properties.RefreshToken)
			if loginErr != nil {
				_, loginErr = tb.login(tb.properties.Username, tb.properties.Password)
				if loginErr != nil {
					return loginErr
				}
			}
		} else {
			_, loginErr := tb.login(tb.properties.Username, tb.properties.Password)
			if loginErr != nil {
				return loginErr
			}
		}
	}

	tb.downloadClient = req.C().SetCommonHeader(HeaderUserAgent, DownloadUserAgent)

	err = tb.WriteConfig(tb.properties)
	if err != nil {
		return err
	}
	return nil
}

func (tb *ThunderBrowser) Drop() error {
	return pan.OnlyMsg("drop not support")
}

func (tb *ThunderBrowser) Disk() (*pan.DiskResp, error) {
	about, err := tb.about()
	if err != nil {
		return nil, err
	}
	total, _ := strconv.ParseInt(about.Quota.Limit, 10, 64)
	usage, _ := strconv.ParseInt(about.Quota.Usage, 10, 64)
	return &pan.DiskResp{
		Total: total / 1024 / 1024,
		Free:  (total - usage) / 1024 / 1024,
		Used:  usage / 1024 / 1024,
		Ext: map[string]interface{}{
			QuotaCreateOfflineTaskLimit: about.Quotas[QuotaCreateOfflineTaskLimit],
		},
	}, nil
}
func (tb *ThunderBrowser) List(req pan.ListReq) ([]*pan.PanObj, error) {
	queryDir := req.Dir
	if queryDir.Path == "/" && queryDir.Name == "" {
		queryDir.Id = "0"
	}
	if queryDir.Id == "" {
		obj, err := tb.GetPanObj(strings.TrimRight(queryDir.Path, "/")+"/"+queryDir.Name, true, tb.List)
		if err != nil {
			return nil, err
		}
		queryDir = obj
	}
	cacheKey := cacheDirectoryPrefix + queryDir.Id
	if req.Reload {
		tb.Del(cacheKey)
	}
	panObjs, exist, err := tb.GetOrDefault(cacheKey, func() (interface{}, error) {
		files, e := tb.getFiles(queryDir.Id)
		if e != nil {
			logger.Error(e)
			return nil, e
		}
		panObjs := make([]*pan.PanObj, 0)
		for _, item := range files {
			fileType := "file"
			if item.Kind == "drive#folder" {
				fileType = "dir"
			}
			path := strings.TrimRight(queryDir.Path, "/") + "/" + queryDir.Name
			if queryDir.Id == "" {
				path = "/"
			}
			size, _ := strconv.ParseInt(item.Size, 10, 64)
			panObjs = append(panObjs, &pan.PanObj{
				Id:     item.ID,
				Name:   item.Name,
				Path:   path,
				Size:   size,
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
func (tb *ThunderBrowser) ObjRename(req pan.ObjRenameReq) error {
	if req.Obj.Id == "0" || (req.Obj.Path == "/" && req.Obj.Name == "") {
		return pan.OnlyMsg("not support rename root path")
	}
	object := req.Obj
	if object.Id == "" {
		path := strings.Trim(req.Obj.Path, "/") + "/" + req.Obj.Name
		obj, err := tb.GetPanObj(path, true, tb.List)
		if err != nil {
			return err
		}
		object = obj
	}
	newFile, err := tb.rename(object.Id, req.NewName)
	if err != nil {
		return err
	}
	tb.Del(cacheDirectoryPrefix + newFile.ParentID)
	return nil
}
func (tb *ThunderBrowser) BatchRename(req pan.BatchRenameReq) error {
	objs, err := tb.List(pan.ListReq{
		Reload: true,
		Dir:    req.Path,
	})
	if err != nil {
		return err
	}
	for _, object := range objs {
		if object.Type == "dir" {
			err = tb.BatchRename(pan.BatchRenameReq{
				Path: object,
				Func: req.Func,
			})
			if err != nil {
				return err
			}
		}
		newName := req.Func(object)

		if newName != object.Name {
			err = tb.ObjRename(pan.ObjRenameReq{
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
func (tb *ThunderBrowser) Mkdir(req pan.MkdirReq) (*pan.PanObj, error) {
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
	obj, err := tb.GetPanObj(targetPath, false, tb.List)
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
			resp, err := tb.makeDir(s, targetDirId)
			if err != nil {
				return nil, pan.OnlyError(err)
			}
			// 这里有问题
			targetDirId = resp.File.ID
		}
		tb.Del(cacheDirectoryPrefix + obj.Id)
		return tb.Mkdir(req)
	}
}
func (tb *ThunderBrowser) Move(req pan.MovieReq) error {
	targetObj := req.TargetObj
	if targetObj.Type == "file" {
		return pan.OnlyMsg("target is a file")
	}
	// 重新直接创建目标目录
	if targetObj.Id == "" {
		create, err := tb.Mkdir(pan.MkdirReq{
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
			obj, err := tb.GetPanObj(item.Path, true, tb.List)
			if err == nil {
				objIds = append(objIds, obj.Id)
				if obj.Type == "dir" {
					reloadDirId[obj.Id] = true
				}
			}
		}
	}
	err := tb.move(objIds, targetObj.Id)
	if err != nil {
		return pan.OnlyError(err)
	}
	for key, _ := range reloadDirId {
		tb.Del(cacheDirectoryPrefix + key)
	}
	return nil
}
func (tb *ThunderBrowser) Delete(req pan.DeleteReq) error {
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
			obj, err := tb.GetPanObj(item.Path, true, tb.List)
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
		err := tb.remove(objIds)
		if err != nil {
			return err
		}
		for key, _ := range reloadDirId {
			tb.Del(cacheDirectoryPrefix + key)
		}
	}

	return nil
}

func (tb *ThunderBrowser) UploadPath(req pan.UploadPathReq) error {
	return tb.BaseUploadPath(req, tb.UploadFile)
}

func (tb *ThunderBrowser) UploadFile(req pan.UploadFileReq) error {
	if req.Resumable {
		logger.Warn("thunder_browser is not support resumeable")
	}
	if req.OnlyFast {
		return pan.OnlyMsg("thunder_browser is not support fast upload")
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
	_, err = tb.GetPanObj(remoteAllPath, true, tb.List)
	// 没有报错证明文件已经存在
	if err == nil {
		return pan.CodeMsg(CodeObjectExist, remoteAllPath+" is exist")
	}
	dir, err := tb.Mkdir(pan.MkdirReq{
		NewPath: remotePath,
	})
	if err != nil {
		return pan.MsgError(remotePath+" create error", err)
	}

	gcid, err := internal.GetFileGcid(req.LocalFile)
	if err != nil {
		return err
	}
	parentId := dir.Id
	if parentId == "0" {
		parentId = ""
	}
	resp, err := tb.uploadTask(UploadTaskRequest{
		Kind:       FILE,
		ParentId:   parentId,
		Name:       remoteName,
		Size:       stat.Size(),
		Hash:       gcid,
		UploadType: UploadTypeResumable,
		Space:      ThunderDriveSpace,
	})

	if err != nil {
		return err
	}

	param := resp.Resumable.Params
	if resp.UploadType == UploadTypeResumable {
		param.Endpoint = strings.TrimLeft(param.Endpoint, param.Bucket+".")
		s, err := session.NewSession(&aws.Config{
			Credentials: credentials.NewStaticCredentials(param.AccessKeyID, param.AccessKeySecret, param.SecurityToken),
			Region:      aws.String("xunlei"),
			Endpoint:    aws.String(param.Endpoint),
		})
		if err != nil {
			return err
		}
		uploader := s3manager.NewUploader(s)
		if stat.Size() > s3manager.MaxUploadParts*s3manager.DefaultUploadPartSize {
			uploader.PartSize = stat.Size() / (s3manager.MaxUploadParts - 1)
		}
		file, err := os.Open(req.LocalFile)
		if err != nil {
			return err
		}
		_, err = uploader.Upload(&s3manager.UploadInput{
			Bucket:  aws.String(param.Bucket),
			Key:     aws.String(param.Key),
			Expires: aws.Time(param.Expiration),
			Body:    io.TeeReader(file, pan.NewProgressWriter(req.LocalFile, stat.Size())),
		})
		_ = file.Close()
		if err == nil && req.SuccessDel {
			err = os.Remove(req.LocalFile)
			if err != nil {
				logger.Errorf("delete fail %s,%v", req.LocalFile, err)
			} else {
				logger.Infof("delete success %s", req.LocalFile)
			}
		}
		return err
	}

	return nil
}

func (tb *ThunderBrowser) DownloadPath(req pan.DownloadPathReq) error {
	return tb.BaseDownloadPath(req, tb.List, tb.DownloadFile)
}
func (tb *ThunderBrowser) DownloadFile(req pan.DownloadFileReq) error {
	return tb.BaseDownloadFile(req, tb.downloadClient, func(req pan.DownloadFileReq) (string, error) {
		link, err := tb.getLink(req.RemoteFile.Id)
		if err != nil {
			return "", err
		}
		downloadLink := link.WebContentLink
		if downloadLink == "" {
			logger.Errorf("cant get link:%s,try media link", req.RemoteFile.Name)
			for _, media := range link.Medias {
				if media.Link.URL != "" {
					downloadLink = media.Link.URL
					break
				}
			}
		}
		if downloadLink == "" {
			logger.Debugf("cant get link:%s,%v", req.RemoteFile.Name, link)
			return "", pan.OnlyMsg(fmt.Sprintf("cant get link:%s", req.RemoteFile.Name))
		}
		return downloadLink, nil
	})
}

func (tb *ThunderBrowser) OfflineDownload(req pan.OfflineDownloadReq) (*pan.Task, error) {
	dir, err := tb.Mkdir(pan.MkdirReq{
		NewPath: req.RemotePath,
	})
	if err != nil {
		return nil, pan.MsgError(req.RemotePath+" create error", err)
	}
	parentId := dir.Id
	if parentId == "0" {
		parentId = ""
	}
	remoteName := req.RemoteName
	if remoteName == "" {
		remoteName = req.Url
	}
	taskResp, e := tb.uploadTask(UploadTaskRequest{
		Kind:       FILE,
		ParentId:   parentId,
		Name:       remoteName,
		UploadType: UploadTypeUrl,
		Space:      ThunderDriveSpace,
		Url: Url{
			Url:   req.Url,
			Files: []string{},
		},
	})

	if e != nil {
		return nil, e
	}
	task := taskResp.Task
	return &pan.Task{
		Id:          task.Id,
		Name:        task.Name,
		Type:        task.Type,
		Phase:       task.Phase,
		CreatedTime: task.CreatedTime.Time,
		UpdatedTime: task.UpdatedTime.Time,
	}, nil
}

func (tb *ThunderBrowser) TaskList(req pan.TaskListReq) ([]*pan.Task, error) {
	tasks, err := tb.taskQuery(TaskQueryRequest{
		Space:  ThunderDriveSpace,
		Types:  req.Types,
		Ids:    req.Ids,
		Phases: req.Phases,
		With:   "reference_resource",
		Limit:  100,
	})
	if err != nil {
		return nil, err
	}
	panTasks := make([]*pan.Task, 0)
	for _, task := range tasks {
		panTasks = append(panTasks, &pan.Task{
			Id:          task.Id,
			Name:        task.Name,
			Type:        task.Type,
			Phase:       task.Phase,
			CreatedTime: task.CreatedTime.Time,
			UpdatedTime: task.UpdatedTime.Time,
		})
	}

	return panTasks, nil
}

func (tb *ThunderBrowser) ShareList(req pan.ShareListReq) ([]*pan.ShareData, error) {
	shareList, err := tb.shareList(req.ShareIds...)
	if err != nil {
		return nil, err
	}
	result := make([]*pan.ShareData, 0)
	for _, share := range shareList {
		result = append(result, &pan.ShareData{
			ShareId:  share.ShareId,
			ShareUrl: share.ShareUrl,
			PassCode: share.PassCode,
			Title:    share.Title,
		})
	}
	return result, nil
}
func (tb *ThunderBrowser) NewShare(req pan.NewShareReq) (*pan.ShareData, error) {
	share, err := tb.createShare(CreateShareReq{
		FileIds: req.Fids,
		ShareTo: "copy",
		Params: CreateShareParams{
			SubscribePush:      "false",
			WithPassCodeInLink: strconv.FormatBool(req.NeedPassCode),
		},
		Title:          req.Title,
		RestoreLimit:   "-1",
		ExpirationDays: strconv.Itoa(req.ExpiredType),
	})
	if err != nil {
		return nil, err
	}
	return &pan.ShareData{
		ShareId:  share.ShareId,
		ShareUrl: share.ShareUrl,
		PassCode: share.PassCode,
	}, nil
}
func (tb *ThunderBrowser) DeleteShare(req pan.DelShareReq) error {
	shareIds := req.ShareIds
	for _, shareId := range shareIds {
		err := tb.deleteShare(shareId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (tb *ThunderBrowser) ShareRestore(req pan.ShareRestoreReq) error {
	passCode := req.PassCode
	shareId := req.ShareId
	targetDir := req.TargetDir
	if targetDir == "" {
		targetDir = "/"
	}
	if req.ShareId == "" {
		if req.ShareUrl == "" {
			return pan.OnlyMsg("share url is null")
		}
		// 解析URL
		parsedURL, err := url.Parse(req.ShareUrl)
		if err != nil {
			return err
		}

		// 获取查询参数
		queryParams := parsedURL.Query()
		shareId = strings.TrimLeft(parsedURL.Path, "/s/")
		// 从查询参数中提取分享ID和密码
		passCode = queryParams.Get("pwd")
	}
	parentDir, err := tb.Mkdir(pan.MkdirReq{
		NewPath: targetDir,
	})
	if err != nil {
		return err
	}
	share, err := tb.getShare(ShareDetailReq{
		ShareId:  shareId,
		PassCode: passCode,
	})
	if err != nil {
		return err
	}
	fileIds := make([]string, 0)
	for _, file := range share.Files {
		fileIds = append(fileIds, file.ID)
	}
	restore, err := tb.restore(RestoreReq{
		ParentId:        parentDir.Id,
		ShareId:         shareId,
		PassCodeToken:   share.PassCodeToken,
		AncestorIds:     nil,
		FileIds:         fileIds,
		SpecifyParentId: true,
	})
	if err != nil {
		return err
	}
	for {
		info, err := tb.taskInfo(restore.RestoreTaskId)
		if err != nil {
			return err
		}
		if info.Phase == PhaseTypeComplete {
			break
		}
		time.Sleep(time.Second)
	}
	return nil
}

func (tb *ThunderBrowser) DirectLink(req pan.DirectLinkReq) ([]*pan.DirectLink, error) {
	return nil, pan.OnlyMsg("direct link not support")
}

func init() {
	pan.RegisterDriver(pan.ThunderBrowser, func() pan.Driver {
		return &ThunderBrowser{
			PropertiesOperate: pan.PropertiesOperate{
				DriverType: pan.ThunderBrowser,
			},
			CacheOperate:  pan.CacheOperate{DriverType: pan.ThunderBrowser},
			CommonOperate: pan.CommonOperate{},
		}
	})
}

func BuildCustomUserAgent(appName, sdkVersion, clientVersion string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("ANDROID-%s/%s ", appName, clientVersion))
	sb.WriteString("networkType/WIFI ")
	sb.WriteString(fmt.Sprintf("appid/%s ", "22062"))
	sb.WriteString(fmt.Sprintf("deviceName/Xiaomi_M2004j7ac "))
	sb.WriteString(fmt.Sprintf("deviceModel/M2004J7AC "))
	sb.WriteString(fmt.Sprintf("OSVersion/13 "))
	sb.WriteString(fmt.Sprintf("protocolVersion/301 "))
	sb.WriteString(fmt.Sprintf("platformversion/10 "))
	sb.WriteString(fmt.Sprintf("sdkVersion/%s ", sdkVersion))
	sb.WriteString(fmt.Sprintf("Oauth2Client/0.9 (Linux 4_9_337-perf-sn-uotan-gd9d488809c3d) (JAVA 0) "))
	return sb.String()
}
