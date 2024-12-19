package pan

import "time"

type Properties interface {
	// OnlyImportProperties 仅仅定义一个接口来进行继承
	OnlyImportProperties()
}

type Json map[string]interface{}

type DiskResp struct {
	Used  int64
	Free  int64
	Total int64
	Ext   Json
}

type ListReq struct {
	Reload bool
	Dir    *PanObj
}

type MkdirReq struct {
	NewPath string
	Parent  *PanObj
}

type DeleteReq struct {
	Items []*PanObj
}

type MovieReq struct {
	Items     []*PanObj
	TargetObj *PanObj
}

type ObjRenameReq struct {
	Obj     *PanObj
	NewName string
}

type BatchRenameFunc func(obj *PanObj) string
type BatchRenameReq struct {
	Path *PanObj
	Func BatchRenameFunc
}

type PanObj struct {
	Id   string
	Name string
	Path string
	Size int64
	Type string
	// 额外的数据
	Ext    Json
	Parent *PanObj
}
type RemoteTransfer func(remote string) string

type UploadFileReq struct {
	LocalFile          string
	RemotePath         string
	OnlyFast           bool
	Resumable          bool
	SuccessDel         bool
	RemotePathTransfer RemoteTransfer
	RemoteNameTransfer RemoteTransfer
}

type UploadPathReq struct {
	LocalPath          string
	RemotePath         string
	Resumable          bool
	SkipFileErr        bool
	SuccessDel         bool
	OnlyFast           bool
	IgnorePaths        []string
	IgnoreFiles        []string
	Extensions         []string
	IgnoreExtensions   []string
	RemotePathTransfer RemoteTransfer
	RemoteNameTransfer RemoteTransfer
}
type DownloadCallback func(localPath, localFile string)

type DownloadPathReq struct {
	RemotePath         *PanObj
	LocalPath          string
	Concurrency        int
	ChunkSize          int64
	OverCover          bool
	SkipFileErr        bool
	IgnorePaths        []string
	IgnoreFiles        []string
	Extensions         []string
	IgnoreExtensions   []string
	RemoteNameTransfer RemoteTransfer
	DownloadCallback
}

type DownloadFileReq struct {
	RemoteFile  *PanObj
	LocalPath   string
	Concurrency int
	ChunkSize   int64
	OverCover   bool
	DownloadCallback
}

type OfflineDownloadReq struct {
	RemotePath string
	RemoteName string
	Url        string
}

type TaskListReq struct {
	Ids    []string
	Name   string
	Types  []string
	Phases []string
}

type Task struct {
	Id          string
	Name        string
	Type        string
	Phase       string
	CreatedTime time.Time
	UpdatedTime time.Time
	Ext         Json
}

type DelShareReq struct {
	ShareIds []string
}

type NewShareReq struct {
	// 分享的文件ID
	Fids []string
	// 分享标题
	Title string
	// 需要密码,thunder 无效
	NeedPassCode bool
	// quark 1 无限期 2 1天 3 7天 4 30天
	// thunder -1 不限 1 1天 2 2天 3 3天 4 4天 如此类推
	ExpiredType int
}

type ShareData struct {
	ShareUrl string
	ShareId  string
	Title    string
	PassCode string
	Ext      Json
}

type ShareListReq struct {
	ShareIds []string
}

type ShareRestoreReq struct {
	// 分享的具体链接，带pwd
	ShareUrl string
	// 下面的有值会优先处理
	ShareId  string
	PassCode string
	// 保存的目录
	TargetDir string
}
