package pan_client

import (
	"fmt"
	"github.com/hefeiyu2025/pan-client/pan"
	"testing"
)

func Test(t *testing.T) {
	//defer GracefulExist()
	client, err := GetClient(pan.Quark)
	if err != nil {
		t.Error(err)
		return
	}

	list, err := client.List(pan.ListReq{Dir: &pan.PanObj{
		Id: "0",
	}})
	if err != nil {
		t.Error(err)
		return
	}
	for _, item := range list {
		if item.Type == "file" {
			err = client.DownloadFile(pan.DownloadFileReq{
				RemoteFile: item,
				LocalPath:  "./tmp",
				OverCover:  true,
				DownloadCallback: func(localPath, localFile string) {
					fmt.Print(localPath, localFile)
				},
			})
			if err != nil {
				t.Error(err)
				return
			}
		}
	}
}
