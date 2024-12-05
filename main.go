package main

import (
	"github.com/hefeiyu2025/pan-client/client"
	_ "github.com/hefeiyu2025/pan-client/client/driver"
	"github.com/hefeiyu2025/pan-client/common"
)

func main() {
	defer common.Exit()
	driver, err := client.GetDriver(client.Cloudreve)
	if err != nil {
		panic(err)
	}
	err = driver.UploadPath(client.OneStepUploadPathReq{
		LocalPath:   "D:/download/170",
		RemotePath:  "/demo1",
		Resumable:   true,
		SkipFileErr: true,
		SuccessDel:  false,
	})
	if err != nil {
		panic(err)
	}
}
