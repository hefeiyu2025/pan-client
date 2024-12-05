package main

import (
	"github.com/hefeiyu2025/pan-client/client"
	_ "github.com/hefeiyu2025/pan-client/client/driver"
	"github.com/hefeiyu2025/pan-client/common"
	logger "github.com/sirupsen/logrus"
)

func main() {
	driver, err := client.GetDriver(client.Cloudreve)
	if err != nil {
		panic(err)
	}
	logger.Info(driver)

	defer common.Exit()
}
