package common

import (
	"encoding/gob"
	"github.com/hefeiyu2025/pan-client/internal"
	"github.com/hefeiyu2025/pan-client/pan"
	"github.com/hefeiyu2025/pan-client/pan/driver/cloudreve"
)

func init() {
	internal.InitConfig()
	internal.InitLog()
	InitGob()
	internal.InitCache()
	internal.InitChunkDownload()
	InitExitHook()
}
func InitGob() {
	gob.Register(&cloudreve.PolicySummary{})
	gob.Register([]*pan.PanObj{})
	gob.Register(&pan.PanObj{})
	//gob.RegisterName("*cloudreve.UploadCredential", &cloudreve.UploadCredential{})
	gob.RegisterName("cloudreve.UploadCredential", cloudreve.UploadCredential{})
}
