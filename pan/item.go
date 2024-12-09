package pan

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
	Id     string
	Name   string
	Path   string
	Size   int64
	Type   string
	Parent *PanObj
}

type UploadFileReq struct {
	LocalFile      string
	RemotePath     string
	Resumable      bool
	SuccessDel     bool
	RemoteTransfer func(remotePath, remoteName string) (string, string)
}

type UploadPathReq struct {
	LocalPath        string
	RemotePath       string
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

type DownloadPathReq struct {
	RemotePath       *PanObj
	LocalPath        string
	Concurrency      int
	ChunkSize        int64
	OverCover        bool
	SkipFileErr      bool
	DownloadCallback DownloadCallback
}

type DownloadFileReq struct {
	RemoteFile       *PanObj
	LocalPath        string
	Concurrency      int
	ChunkSize        int64
	OverCover        bool
	DownloadCallback DownloadCallback
}
