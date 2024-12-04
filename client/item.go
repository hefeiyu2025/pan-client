package client

type Properties interface {
	// OnlyImportProperties 仅仅定义一个接口来进行继承
	OnlyImportProperties()
}

type DiskResp struct {
	Used  uint64 `json:"used"`
	Free  uint64 `json:"free"`
	Total uint64 `json:"total"`
}

type ListReq struct {
	Reload bool
	Dir    *PanObj
}

type PanObj struct {
	Id     string
	Name   string
	Path   string
	Size   uint64
	Type   string
	Parent *PanObj
}

type OneStepUploadFileReq struct {
	LocalFile      string
	RemotePath     string
	PolicyId       string
	Resumable      bool
	SuccessDel     bool
	RemoteTransfer func(remotePath, remoteName string) (string, string)
}

type OneStepUploadPathReq struct {
	LocalPath        string
	RemotePath       string
	PolicyId         string
	Resumable        bool
	SkipFileErr      bool
	SuccessDel       bool
	IgnorePaths      []string
	IgnoreFiles      []string
	Extensions       []string
	IgnoreExtensions []string
	RemoteTransfer   func(remotePath, remoteName string) (string, string)
}
type DownloadCallback func(localPath, localFile string)

type OneStepDownloadPathReq struct {
	Remote           string
	LocalPath        string
	IsParallel       bool
	SegmentSize      int64
	DownloadCallback DownloadCallback
}

type OneStepDownloadFileReq struct {
	Remote           string
	LocalPath        string
	IsParallel       bool
	SegmentSize      int64
	DownloadCallback DownloadCallback
}
