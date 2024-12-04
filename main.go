package main

import (
	"github.com/hefeiyu2025/pan-client/client"
	_ "github.com/hefeiyu2025/pan-client/client/driver"
	logger "github.com/sirupsen/logrus"
)

func main() {
	driver, err := client.GetDriver(client.Cloudreve)
	if err != nil {
		panic(err)
	}
	data, err := driver.List(client.ListReq{
		Reload: false,
		Dir: &client.PanObj{
			Path: "/",
		},
	})
	if err != nil {
		panic(err)
	}
	logger.Info(data)
	data1, err := driver.List(client.ListReq{
		Reload: false,
		Dir: &client.PanObj{
			Path: "/",
		},
	})
	if err != nil {
		panic(err)
	}
	logger.Info(data1)
}
