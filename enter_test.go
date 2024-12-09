package pan_client

import (
	"github.com/hefeiyu2025/pan-client/pan"
	logger "github.com/sirupsen/logrus"
	"testing"
)

func TestDownloadAndUpload(t *testing.T) {
	//defer GracefulExist()
	client, err := GetClient(pan.Cloudreve)
	if err != nil {
		t.Error(err)
		return
	}

	list, err := client.List(pan.ListReq{Dir: &pan.PanObj{
		Id: "0",
	}, Reload: true})
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
					logger.Info(localPath, localFile)
				},
			})
			if err != nil {
				t.Error(err)
				return
			}
			err = client.ObjRename(pan.ObjRenameReq{
				Obj:     item,
				NewName: "2.pdf",
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
					Name: "test",
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
				LocalPath:   "./tmp",
				RemotePath:  "/",
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
	_, err = client.List(pan.ListReq{Dir: &pan.PanObj{
		Id: "0",
	}, Reload: true})
	if err != nil {
		t.Error(err)
		return
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
