package pan

import "time"

type Json map[string]interface{}

type DiskResp struct {
	Used  int64 `json:"used,omitempty"`
	Free  int64 `json:"free,omitempty"`
	Total int64 `json:"total,omitempty"`
	Ext   Json  `json:"ext,omitempty"`
}

type ListReq struct {
	Reload bool    `json:"reload,omitempty"`
	Dir    *PanObj `json:"dir,omitempty"`
}

type MkdirReq struct {
	NewPath string  `json:"newPath,omitempty"`
	Parent  *PanObj `json:"parent,omitempty"`
}

type DeleteReq struct {
	Items []*PanObj `json:"items,omitempty"`
}

type MovieReq struct {
	Items     []*PanObj `json:"items,omitempty"`
	TargetObj *PanObj   `json:"targetObj,omitempty"`
}

type ObjRenameReq struct {
	Obj     *PanObj `json:"obj,omitempty"`
	NewName string  `json:"newName,omitempty"`
}

type BatchRenameFunc func(obj *PanObj) string
type BatchRenameReq struct {
	Path *PanObj `json:"path,omitempty"`
	Func BatchRenameFunc
}

type PanObj struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
	Type string `json:"type"`
	// 额外的数据
	Ext    Json    `json:"ext"`
	Parent *PanObj `json:"parent"`
}
type RemoteTransfer func(remote string) string

type UploadFileReq struct {
	LocalFile          string `json:"localFile,omitempty"`
	RemotePath         string `json:"remotePath,omitempty"`
	OnlyFast           bool   `json:"onlyFast,omitempty"`
	Resumable          bool   `json:"resumable,omitempty"`
	SuccessDel         bool   `json:"successDel,omitempty"`
	RemotePathTransfer RemoteTransfer
	RemoteNameTransfer RemoteTransfer
}

type UploadPathReq struct {
	LocalPath          string   `json:"localPath,omitempty"`
	RemotePath         string   `json:"remotePath,omitempty"`
	Resumable          bool     `json:"resumable,omitempty"`
	SkipFileErr        bool     `json:"skipFileErr,omitempty"`
	SuccessDel         bool     `json:"successDel,omitempty"`
	OnlyFast           bool     `json:"onlyFast,omitempty"`
	IgnorePaths        []string `json:"ignorePaths,omitempty"`
	IgnoreFiles        []string `json:"ignoreFiles,omitempty"`
	Extensions         []string `json:"extensions,omitempty"`
	IgnoreExtensions   []string `json:"ignoreExtensions,omitempty"`
	RemotePathTransfer RemoteTransfer
	RemoteNameTransfer RemoteTransfer
}
type DownloadCallback func(localPath, localFile string)

type DownloadPathReq struct {
	RemotePath         *PanObj  `json:"remotePath,omitempty"`
	LocalPath          string   `json:"localPath,omitempty"`
	Concurrency        int      `json:"concurrency,omitempty"`
	ChunkSize          int64    `json:"chunkSize,omitempty"`
	OverCover          bool     `json:"overCover,omitempty"`
	SkipFileErr        bool     `json:"skipFileErr,omitempty"`
	IgnorePaths        []string `json:"ignorePaths,omitempty"`
	IgnoreFiles        []string `json:"ignoreFiles,omitempty"`
	Extensions         []string `json:"extensions,omitempty"`
	IgnoreExtensions   []string `json:"ignoreExtensions,omitempty"`
	RemoteNameTransfer RemoteTransfer
	DownloadCallback
}

type DownloadFileReq struct {
	RemoteFile       *PanObj `json:"remoteFile,omitempty"`
	LocalPath        string  `json:"localPath,omitempty"`
	Concurrency      int     `json:"concurrency,omitempty"`
	ChunkSize        int64   `json:"chunkSize,omitempty"`
	OverCover        bool    `json:"overCover,omitempty"`
	DownloadCallback `json:"downloadCallback,omitempty"`
}

type OfflineDownloadReq struct {
	RemotePath string `json:"remotePath,omitempty"`
	RemoteName string `json:"remoteName,omitempty"`
	Url        string `json:"url,omitempty"`
}

type TaskListReq struct {
	Ids    []string `json:"ids,omitempty"`
	Name   string   `json:"name,omitempty"`
	Types  []string `json:"types,omitempty"`
	Phases []string `json:"phases,omitempty"`
}

type Task struct {
	Id          string    `json:"id,omitempty"`
	Name        string    `json:"name,omitempty"`
	Type        string    `json:"type,omitempty"`
	Phase       string    `json:"phase,omitempty"`
	CreatedTime time.Time `json:"createdTime"`
	UpdatedTime time.Time `json:"updatedTime"`
	Ext         Json      `json:"ext,omitempty"`
}

type DelShareReq struct {
	ShareIds []string `json:"shareIds,omitempty"`
}

type NewShareReq struct {
	// 分享的文件ID
	Fids []string `json:"fids,omitempty"`
	// 分享标题
	Title string `json:"title,omitempty"`
	// 需要密码,thunder 无效
	NeedPassCode bool `json:"needPassCode,omitempty"`
	// quark 1 无限期 2 1天 3 7天 4 30天
	// thunder -1 不限 1 1天 2 2天 3 3天 4 4天 如此类推
	ExpiredType int `json:"expiredType,omitempty"`
}

type ShareData struct {
	ShareUrl string `json:"shareUrl,omitempty"`
	ShareId  string `json:"shareId,omitempty"`
	Title    string `json:"title,omitempty"`
	PassCode string `json:"passCode,omitempty"`
	Ext      Json   `json:"ext,omitempty"`
}

type ShareListReq struct {
	ShareIds []string `json:"shareIds,omitempty"`
}

type ShareRestoreReq struct {
	// 分享的具体链接，带pwd
	ShareUrl string `json:"shareUrl,omitempty"`
	// 下面的有值会优先处理
	ShareId  string `json:"shareId,omitempty"`
	PassCode string `json:"passCode,omitempty"`
	// 保存的目录
	TargetDir string `json:"targetDir,omitempty"`
}
type DirectLinkReq struct {
	List []*DirectLink `json:"list"`
}

type DirectLink struct {
	FileId string `json:"fileId,omitempty"`
	Name   string `json:"name,omitempty"`
	Link   string `json:"link,omitempty"`
}
