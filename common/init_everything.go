package common

import (
	"encoding/gob"
	"github.com/hefeiyu2025/pan-client/client"
	"github.com/hefeiyu2025/pan-client/client/driver/cloudreve"
	"github.com/hefeiyu2025/pan-client/internal"
)

func init() {
	internal.InitConfig()
	internal.InitLog()
	InitGob()
	internal.InitCache()
	InitExitHook()
}
func InitGob() {
	gob.Register(&cloudreve.PolicySummary{})
	gob.Register([]*client.PanObj{})
	gob.Register(&client.PanObj{})
	//gob.RegisterName("*cloudreve.UploadCredential", &cloudreve.UploadCredential{})
	gob.RegisterName("cloudreve.UploadCredential", cloudreve.UploadCredential{})
}
