package pan_client

import (
	"fmt"
	"github.com/hefeiyu2025/pan-client/pan"
	logger "github.com/sirupsen/logrus"
	"testing"
)

func TestDownloadAndUpload(t *testing.T) {
	defer GracefulExist()
	client, err := GetClient(pan.ThunderBrowser)
	if err != nil {
		t.Error(err)
		return
	}

	list, err := client.List(pan.ListReq{Dir: &pan.PanObj{
		Path: "/",
		Name: "test1",
	}, Reload: true})
	if err != nil {
		t.Error(err)
		return
	}
	for _, item := range list {
		if item.Type == "file" {
			err = client.DownloadFile(pan.DownloadFileReq{
				RemoteFile:  item,
				LocalPath:   "./tmpdata",
				ChunkSize:   50 * 1024 * 1024,
				OverCover:   false,
				Concurrency: 2,
				DownloadCallback: func(localPath, localFile string) {
					logger.Info(localPath, localFile)
				},
			})
			if err != nil {
				t.Error(err)
				return
			}
			err = client.ObjRename(pan.ObjRenameReq{
				Obj:     item,
				NewName: "1.pdf",
			})
			if err != nil {
				t.Error(err)
				return
			}
			err = client.ObjRename(pan.ObjRenameReq{
				Obj:     item,
				NewName: "后浪电影学院039《看不见的剪辑》.pdf",
			})
			if err != nil {
				t.Error(err)
				return
			}
			err = client.Move(pan.MovieReq{
				Items: []*pan.PanObj{item},
				TargetObj: &pan.PanObj{
					Name: "test2",
					Path: "/",
					Type: "dir",
				},
			})
			if err != nil {
				t.Error(err)
				return
			}

			err = client.Delete(pan.DeleteReq{
				Items: []*pan.PanObj{item},
			})
			if err != nil {
				t.Error(err)
				return
			}
			err = client.UploadPath(pan.UploadPathReq{
				LocalPath:   "./tmpdata",
				RemotePath:  "/test1",
				Resumable:   true,
				SkipFileErr: false,
				SuccessDel:  true,
				Extensions:  []string{".pdf"},
			})
			if err != nil {
				t.Error(err)
				return
			}
		}
	}
}

func TestUpload(t *testing.T) {
	//defer GracefulExist()
	client, err := GetClient(pan.Quark)
	if err != nil {
		t.Error(err)
		return
	}
	objs, err := client.List(pan.ListReq{Dir: &pan.PanObj{
		Path: "/来自：分享",
		Name: "BY.4k",
	}, Reload: true})
	if err != nil {
		t.Error(err)
		return
	}
	for _, obj := range objs {
		fmt.Println(obj.Name)
	}

	err = client.UploadPath(pan.UploadPathReq{
		LocalPath:   "D:/download/jdk",
		RemotePath:  "/jdk",
		Resumable:   true,
		SkipFileErr: true,
		SuccessDel:  false,
	})
	if err != nil {
		panic(err)
		return
	}
}
