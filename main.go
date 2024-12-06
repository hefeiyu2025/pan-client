package main

import (
	"github.com/hefeiyu2025/pan-client/client"
	_ "github.com/hefeiyu2025/pan-client/client/driver"
	"github.com/hefeiyu2025/pan-client/common"
	logger "github.com/sirupsen/logrus"
)

func main() {
	defer common.Exit()
	driver, err := client.GetDriver(client.Cloudreve)
	if err != nil {
		panic(err)
	}
	err = driver.DownloadPath(client.OneStepDownloadPathReq{
		RemotePath: &client.PanObj{
			Name: "再見枕邊人6",
			Path: "/影视",
			Type: "dir",
		},
		LocalPath: "./",
		ChunkSize: 20 * 1024 * 1024,
		DownloadCallback: func(localPath, localFile string) {
			logger.Info(localPath, localFile)
		},
	})
	if err != nil {
		panic(err)
	}
}
