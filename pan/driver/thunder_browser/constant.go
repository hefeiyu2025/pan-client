package thunder_browser

const (
	cacheDirectoryPrefix = "directory_"
)

const (
	API_URL        = "https://x-api-pan.xunlei.com/drive/v1"
	XLUSER_API_URL = "https://xluser-ssl.xunlei.com/v1"
)

const (
	ClientID          = "ZUBzD9J_XPXfn7f7"
	ClientSecret      = "yESVmHecEe6F0aou69vl-g"
	ClientVersion     = "1.10.0.2633"
	PackageName       = "com.xunlei.browser"
	HeaderUserAgent   = "User-Agent"
	DownloadUserAgent = "AndroidDownloadManager/13 (Linux; U; Android 13; M2004J7AC Build/SP1A.210812.016)"
	SdkVersion        = "233100"
)

var Algorithms = []string{
	"uWRwO7gPfdPB/0NfPtfQO+71",
	"F93x+qPluYy6jdgNpq+lwdH1ap6WOM+nfz8/V",
	"0HbpxvpXFsBK5CoTKam",
	"dQhzbhzFRcawnsZqRETT9AuPAJ+wTQso82mRv",
	"SAH98AmLZLRa6DB2u68sGhyiDh15guJpXhBzI",
	"unqfo7Z64Rie9RNHMOB",
	"7yxUdFADp3DOBvXdz0DPuKNVT35wqa5z0DEyEvf",
	"RBG",
	"ThTWPG5eC0UBqlbQ+04nZAptqGCdpv9o55A",
}

const (
	ThunderDriveSpace                 = ""
	ThunderDriveSafeSpace             = "SPACE_SAFE"
	ThunderBrowserDriveSpace          = "SPACE_BROWSER"
	ThunderBrowserDriveSafeSpace      = "SPACE_BROWSER_SAFE"
	ThunderDriveFolderType            = "DEFAULT_ROOT"
	ThunderBrowserDriveSafeFolderType = "BROWSER_SAFE"
)

const (
	FOLDER    = "drive#folder"
	FILE      = "drive#file"
	RESUMABLE = "drive#resumable"
	FILELIST  = "drive#fileList"
	TASK      = "drive#task"
)

const (
	UPLOAD_TYPE_UNKNOWN = "UPLOAD_TYPE_UNKNOWN"
	//UPLOAD_TYPE_FORM      = "UPLOAD_TYPE_FORM"
	UPLOAD_TYPE_RESUMABLE = "UPLOAD_TYPE_RESUMABLE"
	UPLOAD_TYPE_URL       = "UPLOAD_TYPE_URL"
)

const (
	PHASE_TYPE_ERROR    = "PHASE_TYPE_ERROR"
	PHASE_TYPE_RUNNING  = "PHASE_TYPE_RUNNING"
	PHASE_TYPE_PENDING  = "PHASE_TYPE_PENDING"
	PHASE_TYPE_COMPLETE = "PHASE_TYPE_COMPLETE"
)

const (
	TASK_TYPE_OFFLINE        = "offline"
	TASK_TYPE_MOVE           = "move"
	TASK_TYPE_UPLOAD         = "upload"
	TASK_TYPE_EVENT_DELETION = "event-deletion"
	TASK_TYPE_DELETEFILE     = "deletefile"
)
