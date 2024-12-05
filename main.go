package main

import (
	"github.com/hefeiyu2025/pan-client/client"
	_ "github.com/hefeiyu2025/pan-client/client/driver"
	"os"
	"strings"
)

func main() {
	defer os.Exit(0)
	driver, err := client.GetDriver(client.Cloudreve)
	if err != nil {
		panic(err)
	}

	err = driver.BatchRename(client.BatchRenameReq{
		Path: &client.PanObj{
			Name: "潜行狙击",
			Path: "/影视",
			Type: "dir",
		},
		Func: func(obj *client.PanObj) string {
			name := obj.Name
			name = strings.Replace(name, ".Lives.of.Omission", "", -1)
			return name
		},
	})

}
