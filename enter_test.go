package pan_client

import (
	"encoding/json"
	"fmt"
	"github.com/hefeiyu2025/pan-client/internal"
	"github.com/hefeiyu2025/pan-client/pan"
	"github.com/hefeiyu2025/pan-client/pan/driver/thunder_browser"
	logger "github.com/sirupsen/logrus"
	"testing"
)

func TestDownloadAndUpload(t *testing.T) {
	defer GracefulExist()
	client, err := GetClient(pan.Cloudreve)
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
		if item.Type == "file" && item.Name == "后浪电影学院039《看不见的剪辑》.pdf" {
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
	defer GracefulExist()
	client, err := GetClient(pan.Cloudreve)
	if err != nil {
		t.Error(err)
		return
	}

	err = client.UploadFile(pan.UploadFileReq{
		LocalFile:  "D:/download/包青天/新包青天/HD高清修復版 _ 新包青天  01_160 _ 情節峰迴路轉扣人心弦 _ 金超群 _ 呂良偉 _ 范鴻軒 _ 曾守明 _粵語_亞視經典劇集_Asia TV Drama_亞視 1995.mp4",
		RemotePath: "/test1",
		Resumable:  true,
		SuccessDel: false,
	})
	if err != nil {
		panic(err)
		return
	}
}

func TestOfflineDownload(t *testing.T) {
	defer GracefulExist()
	client, err := GetClient(pan.ThunderBrowser)
	if err != nil {
		t.Error(err)
		return
	}
	downloadTask, err := client.OfflineDownload(pan.OfflineDownloadReq{
		RemotePath: "/tmpdownload",
		Url:        "magnet:?xt=urn:btih:bd28bedb444fc8293ba86ea8989bfe9e8ff2bf6e&dn=TVBOXNOW+%E6%88%80%E6%84%9B%E8%87%AA%E7%94%B1%E5%BC%8F&tr=udp%3A%2F%2Ftracker.publicbt.com%3A80%2Fannounce&tr=udp%3A%2F%2Ftracker.openbittorrent.com%3A80%2Fannounce&tr=udp%3A%2F%2Fpublic.popcorn-tracker.org%3A6969%2Fannounce&tr=http%3A%2F%2F104.28.1.30%3A8080%2Fannounce&tr=http%3A%2F%2F104.28.16.69%2Fannounce&tr=http%3A%2F%2F107.150.14.110%3A6969%2Fannounce&tr=http%3A%2F%2F109.121.134.121%3A1337%2Fannounce&tr=http%3A%2F%2F114.55.113.60%3A6969%2Fannounce&tr=http%3A%2F%2F125.227.35.196%3A6969%2Fannounce&tr=http%3A%2F%2F128.199.70.66%3A5944%2Fannounce&tr=http%3A%2F%2F157.7.202.64%3A8080%2Fannounce&tr=http%3A%2F%2F158.69.146.212%3A7777%2Fannounce&tr=http%3A%2F%2F173.254.204.71%3A1096%2Fannounce&tr=http%3A%2F%2F178.175.143.27%2Fannounce&tr=http%3A%2F%2F178.33.73.26%3A2710%2Fannounce&tr=http%3A%2F%2F182.176.139.129%3A6969%2Fannounce&tr=http%3A%2F%2F185.5.97.139%3A8089%2Fannounce&tr=http%3A%2F%2F188.165.253.109%3A1337%2Fannounce&tr=http%3A%2F%2F194.106.216.222%2Fannounce&tr=http%3A%2F%2F195.123.209.37%3A1337%2Fannounce&tr=http%3A%2F%2F210.244.71.25%3A6969%2Fannounce&tr=http%3A%2F%2F210.244.71.26%3A6969%2Fannounce&tr=http%3A%2F%2F213.159.215.198%3A6970%2Fannounce&tr=http%3A%2F%2F213.163.67.56%3A1337%2Fannounce&tr=http%3A%2F%2F37.19.5.139%3A6969%2Fannounce&tr=http%3A%2F%2F37.19.5.155%3A6881%2Fannounce&tr=http%3A%2F%2F46.4.109.148%3A6969%2Fannounce&tr=http%3A%2F%2F5.79.249.77%3A6969%2Fannounce&tr=http%3A%2F%2F5.79.83.193%3A2710%2Fannounce&tr=http%3A%2F%2F51.254.244.161%3A6969%2Fannounce&tr=http%3A%2F%2F59.36.96.77%3A6969%2Fannounce&tr=http%3A%2F%2F74.82.52.209%3A6969%2Fannounce&tr=http%3A%2F%2F80.246.243.18%3A6969%2Fannounce&tr=http%3A%2F%2F81.200.2.231%2Fannounce&tr=http%3A%2F%2F85.17.19.180%2Fannounce&tr=http%3A%2F%2F87.248.186.252%3A8080%2Fannounce&tr=http%3A%2F%2F87.253.152.137%2Fannounce&tr=http%3A%2F%2F91.216.110.47%2Fannounce&tr=http%3A%2F%2F91.217.91.21%3A3218%2Fannounce&tr=http%3A%2F%2F91.218.230.81%3A6969%2Fannounce&tr=http%3A%2F%2F93.92.64.5%2Fannounce&tr=http%3A%2F%2Fatrack.pow7.com%2Fannounce&tr=http%3A%2F%2Fbt.henbt.com%3A2710%2Fannounce&tr=http%3A%2F%2Fbt.pusacg.org%3A8080%2Fannounce&tr=http%3A%2F%2Fbt2.careland.com.cn%3A6969%2Fannounce&tr=http%3A%2F%2Fexplodie.org%3A6969%2Fannounce&tr=http%3A%2F%2Fmgtracker.org%3A2710%2Fannounce&tr=http%3A%2F%2Fmgtracker.org%3A6969%2Fannounce&tr=http%3A%2F%2Fopen.acgtracker.com%3A1096%2Fannounce&tr=http%3A%2F%2Fopen.lolicon.eu%3A7777%2Fannounce&tr=http%3A%2F%2Fopen.touki.ru%2Fannounce.php&tr=http%3A%2F%2Fp4p.arenabg.ch%3A1337%2Fannounce&tr=http%3A%2F%2Fp4p.arenabg.com%3A1337%2Fannounce&tr=http%3A%2F%2Fpow7.com%2Fannounce&tr=http%3A%2F%2Fretracker.gorcomnet.ru%2Fannounce&tr=http%3A%2F%2Fretracker.krs-ix.ru%2Fannounce&tr=http%3A%2F%2Fsecure.pow7.com%2Fannounce&tr=http%3A%2F%2Ft1.pow7.com%2Fannounce&tr=http%3A%2F%2Ft2.pow7.com%2Fannounce&tr=http%3A%2F%2Fthetracker.org%2Fannounce&tr=http%3A%2F%2Ftorrent.gresille.org%2Fannounce&tr=http%3A%2F%2Ftorrentsmd.com%3A8080%2Fannounce&tr=http%3A%2F%2Ftracker.aletorrenty.pl%3A2710%2Fannounce&tr=http%3A%2F%2Ftracker.baravik.org%3A6970%2Fannounce&tr=http%3A%2F%2Ftracker.bittor.pw%3A1337%2Fannounce&tr=http%3A%2F%2Ftracker.bittorrent.am%2Fannounce&tr=http%3A%2F%2Ftracker.calculate.ru%3A6969%2Fannounce&tr=http%3A%2F%2Ftracker.dler.org%3A6969%2Fannounce&tr=http%3A%2F%2Ftracker.dutchtracking.com%2Fannounce&tr=http%3A%2F%2Ftracker.dutchtracking.nl%2Fannounce&tr=http%3A%2F%2Ftracker.edoardocolombo.eu%3A6969%2Fannounce&tr=http%3A%2F%2Ftracker.ex.ua%2Fannounce&tr=http%3A%2F%2Ftracker.filetracker.pl%3A8089%2Fannounce&tr=http%3A%2F%2Ftracker.flashtorrents.org%3A6969%2Fannounce&tr=http%3A%2F%2Ftracker.grepler.com%3A6969%2Fannounce&tr=http%3A%2F%2Ftracker.internetwarriors.net%3A1337%2Fannounce&tr=http%3A%2F%2Ftracker.kicks-ass.net%2Fannounce&tr=http%3A%2F%2Ftracker.kuroy.me%3A5944%2Fannounce&tr=http%3A%2F%2Ftracker.mg64.net%3A6881%2Fannounce&tr=http%3A%2F%2Ftracker.opentrackr.org%3A1337%2Fannounce&tr=http%3A%2F%2Ftracker.skyts.net%3A6969%2Fannounce&tr=http%3A%2F%2Ftracker.tfile.me%2Fannounce&tr=http%3A%2F%2Ftracker.tiny-vps.com%3A6969%2Fannounce&tr=http%3A%2F%2Ftracker.tvunderground.org.ru%3A3218%2Fannounce&tr=http%3A%2F%2Ftracker.yoshi210.com%3A6969%2Fannounce&tr=http%3A%2F%2Ftracker1.wasabii.com.tw%3A6969%2Fannounce&tr=http%3A%2F%2Ftracker2.itzmx.com%3A6961%2Fannounce&tr=http%3A%2F%2Ftracker2.wasabii.com.tw%3A6969%2Fannounce&tr=http%3A%2F%2Fwww.wareztorrent.com%2Fannounce&tr=https%3A%2F%2F104.28.17.69%2Fannounce&tr=https%3A%2F%2Fwww.wareztorrent.com%2Fannounce&tr=udp%3A%2F%2F107.150.14.110%3A6969%2Fannounce&tr=udp%3A%2F%2F109.121.134.121%3A1337%2Fannounce&tr=udp%3A%2F%2F114.55.113.60%3A6969%2Fannounce&tr=udp%3A%2F%2F128.199.70.66%3A5944%2Fannounce&tr=udp%3A%2F%2F151.80.120.114%3A2710%2Fannounce&tr=udp%3A%2F%2F168.235.67.63%3A6969%2Fannounce&tr=udp%3A%2F%2F178.33.73.26%3A2710%2Fannounce&tr=udp%3A%2F%2F182.176.139.129%3A6969%2Fannounce&tr=udp%3A%2F%2F185.5.97.139%3A8089%2Fannounce&tr=udp%3A%2F%2F185.86.149.205%3A1337%2Fannounce&tr=udp%3A%2F%2F188.165.253.109%3A1337%2Fannounce&tr=udp%3A%2F%2F191.101.229.236%3A1337%2Fannounce&tr=udp%3A%2F%2F194.106.216.222%3A80%2Fannounce&tr=udp%3A%2F%2F195.123.209.37%3A1337%2Fannounce&tr=udp%3A%2F%2F195.123.209.40%3A80%2Fannounce&tr=udp%3A%2F%2F208.67.16.113%3A8000%2Fannounce&tr=udp%3A%2F%2F213.163.67.56%3A1337%2Fannounce&tr=udp%3A%2F%2F37.19.5.155%3A2710%2Fannounce&tr=udp%3A%2F%2F46.4.109.148%3A6969%2Fannounce&tr=udp%3A%2F%2F5.79.249.77%3A6969%2Fannounce&tr=udp%3A%2F%2F5.79.83.193%3A6969%2Fannounce&tr=udp%3A%2F%2F51.254.244.161%3A6969%2Fannounce&tr=udp%3A%2F%2F62.138.0.158%3A6969%2Fannounce&tr=udp%3A%2F%2F62.212.85.66%3A2710%2Fannounce&tr=udp%3A%2F%2F74.82.52.209%3A6969%2Fannounce&tr=udp%3A%2F%2F85.17.19.180%3A80%2Fannounce&tr=udp%3A%2F%2F89.234.156.205%3A80%2Fannounce&tr=udp%3A%2F%2F9.rarbg.com%3A2710%2Fannounce&tr=udp%3A%2F%2F9.rarbg.me%3A2780%2Fannounce&tr=udp%3A%2F%2F9.rarbg.to%3A2730%2Fannounce&tr=udp%3A%2F%2F91.218.230.81%3A6969%2Fannounce&tr=udp%3A%2F%2F94.23.183.33%3A6969%2Fannounce&tr=udp%3A%2F%2Fbt.xxx-tracker.com%3A2710%2Fannounce&tr=udp%3A%2F%2Feddie4.nl%3A6969%2Fannounce&tr=udp%3A%2F%2Fexplodie.org%3A6969%2Fannounce&tr=udp%3A%2F%2Fmgtracker.org%3A2710%2Fannounce&tr=udp%3A%2F%2Fopen.stealth.si%3A80%2Fannounce&tr=udp%3A%2F%2Fp4p.arenabg.com%3A1337%2Fannounce&tr=udp%3A%2F%2Fshadowshq.eddie4.nl%3A6969%2Fannounce&tr=udp%3A%2F%2Fshadowshq.yi.org%3A6969%2Fannounce&tr=udp%3A%2F%2Ftorrent.gresille.org%3A80%2Fannounce&tr=udp%3A%2F%2Ftracker.aletorrenty.pl%3A2710%2Fannounce&tr=udp%3A%2F%2Ftracker.bittor.pw%3A1337%2Fannounce&tr=udp%3A%2F%2Ftracker.coppersurfer.tk%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker.eddie4.nl%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker.ex.ua%3A80%2Fannounce&tr=udp%3A%2F%2Ftracker.filetracker.pl%3A8089%2Fannounce&tr=udp%3A%2F%2Ftracker.flashtorrents.org%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker.grepler.com%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker.ilibr.org%3A80%2Fannounce&tr=udp%3A%2F%2Ftracker.internetwarriors.net%3A1337%2Fannounce&tr=udp%3A%2F%2Ftracker.kicks-ass.net%3A80%2Fannounce&tr=udp%3A%2F%2Ftracker.kuroy.me%3A5944%2Fannounce&tr=udp%3A%2F%2Ftracker.leechers-paradise.org%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker.mg64.net%3A2710%2Fannounce&tr=udp%3A%2F%2Ftracker.mg64.net%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337%2Fannounce&tr=udp%3A%2F%2Ftracker.piratepublic.com%3A1337%2Fannounce&tr=udp%3A%2F%2Ftracker.sktorrent.net%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker.skyts.net%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker.tiny-vps.com%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker.yoshi210.com%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker2.indowebster.com%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker4.piratux.com%3A6969%2Fannounce&tr=udp%3A%2F%2Fzer0day.ch%3A1337%2Fannounce&tr=udp%3A%2F%2Fzer0day.to%3A1337%2Fannounce",
	})
	if err != nil {
		t.Error(err)
		return
	}
	if downloadTask.Phase != thunder_browser.PhaseTypeComplete {
		taskResp, err := client.TaskList(pan.TaskListReq{
			Ids: []string{downloadTask.Id},
		})
		if err != nil {
			t.Error(err)
			return
		}
		marshal, err := json.Marshal(taskResp)
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Println(string(marshal))
	}
}

func TestShare(t *testing.T) {
	defer GracefulExist()
	client, err := GetClient(pan.Quark)
	if err != nil {
		t.Error(err)
		return
	}
	dir, err := client.Mkdir(pan.MkdirReq{
		NewPath: "/影视/僵",
	})
	if err != nil {
		t.Error(err)
		return
	}
	share, err := client.NewShare(pan.NewShareReq{
		Fids:         []string{dir.Id},
		Title:        "我的分享",
		NeedPassCode: false,
		ExpiredType:  1,
	})
	if err != nil {
		t.Error(err)
		return
	}
	shareList, err := client.ShareList(pan.ShareListReq{
		ShareIds: []string{share.ShareId},
	})
	if err != nil {
		t.Error(err)
		return
	}
	marshal, err := json.Marshal(shareList)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(string(marshal))
	err = client.DeleteShare(pan.DelShareReq{
		ShareIds: []string{share.ShareId},
	})
	if err != nil {
		t.Error(err)
		return
	}
}

func TestShareRestore(t *testing.T) {
	defer GracefulExist()
	client, err := GetClient(pan.Quark)
	if err != nil {
		t.Error(err)
		return
	}
	//err = client.ShareRestore(pan.ShareRestoreReq{
	//	ShareUrl:  "https://pan.xunlei.com/s/VOESxSgsp_Zg1E4WDWxx689sA1?pwd=jab2",
	//	TargetDir: "/tmpdata",
	//})
	//if err != nil {
	//	t.Error(err)
	//	return
	//}
	err = client.ShareRestore(pan.ShareRestoreReq{
		ShareUrl:  "https://pan.quark.cn/s/83dae5e77944",
		PassCode:  "8uSJ",
		TargetDir: "/tmpdata",
	})
	if err != nil {
		t.Error(err)
		return
	}
}

func TestDirectLink(t *testing.T) {
	defer GracefulExist()
	//client, err := GetClient(pan.Cloudreve)
	client, err := GetClientByRw("c3695b6f-6566-400c-bf11-7b08e2c72762", pan.Cloudreve, func(config pan.Properties) error {
		internal.SetDefaultByTag(config)
		return internal.Viper.UnmarshalKey(pan.ViperDriverPrefix+string(pan.Cloudreve), config)

	}, func(config pan.Properties) error {
		internal.Viper.Set(pan.ViperDriverPrefix+string(pan.Cloudreve), config)
		return internal.Viper.WriteConfig()
	})
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
	links := make([]*pan.DirectLink, 0)
	for _, item := range list {
		if item.Type == "file" {
			links = append(links, &pan.DirectLink{
				FileId: item.Id,
				Name:   item.Name,
			})
		}
	}
	link, err := client.DirectLink(pan.DirectLinkReq{List: links})
	if err != nil {
		t.Error(err)
		return
	}
	marshal, err := json.Marshal(link)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(string(marshal))
}
