package internal

import (
	"context"
	"fmt"
	"github.com/imroc/req/v3"
	"io"
	"math"
	urlpkg "net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type ChunkDownload struct {
	url             string
	client          *req.Client
	concurrency     int
	output          io.Writer
	filename        string
	outputDirectory string
	totalBytes      int64
	chunkSize       int64
	tempRootDir     string
	tempDir         string
	taskCh          chan *downloadTask
	doneCh          chan struct{}
	wgDoneCh        chan struct{}
	errCh           chan error
	wg              sync.WaitGroup
	taskMap         map[int]*downloadTask
	taskNotifyCh    chan *downloadTask
	mu              sync.Mutex
	lastIndex       int
}

func NewChunkDownload(url string, client *req.Client) *ChunkDownload {
	return &ChunkDownload{
		url:    url,
		client: client,
	}
}

func (pd *ChunkDownload) completeTask(task *downloadTask) {
	pd.mu.Lock()
	pd.taskMap[task.index] = task
	pd.mu.Unlock()
	go func() {
		select {
		case pd.taskNotifyCh <- task:
		case <-pd.doneCh:
		}
	}()
}

func (pd *ChunkDownload) popTask(index int) *downloadTask {
	pd.mu.Lock()
	if task, ok := pd.taskMap[index]; ok {
		delete(pd.taskMap, index)
		pd.mu.Unlock()
		return task
	}
	pd.mu.Unlock()
	for {
		task := <-pd.taskNotifyCh
		if task.index == index {
			pd.mu.Lock()
			delete(pd.taskMap, index)
			pd.mu.Unlock()
			return task
		}
	}
}

func (pd *ChunkDownload) ensure() error {
	// 若未设置，则仅单线程下载就好
	if pd.concurrency <= 0 {
		pd.concurrency = 1
	}
	if pd.chunkSize <= 0 {
		pd.chunkSize = 1024 * 1024 * 10 // 10MB
	}
	if pd.tempRootDir == "" {
		pd.tempRootDir = os.TempDir()
	}
	pd.tempDir = filepath.Join(pd.tempRootDir, Md5HashStr(pd.url))

	err := os.MkdirAll(pd.tempDir, os.ModePerm)
	if err != nil {
		return err
	}

	pd.taskCh = make(chan *downloadTask)
	pd.doneCh = make(chan struct{})
	pd.wgDoneCh = make(chan struct{})
	pd.errCh = make(chan error)
	pd.taskMap = make(map[int]*downloadTask)
	pd.taskNotifyCh = make(chan *downloadTask)
	return nil
}

func (pd *ChunkDownload) SetChunkSize(chunkSize int64) *ChunkDownload {
	pd.chunkSize = chunkSize
	return pd
}

func (pd *ChunkDownload) SetTempRootDir(tempRootDir string) *ChunkDownload {
	pd.tempRootDir = tempRootDir
	return pd
}

func (pd *ChunkDownload) SetConcurrency(concurrency int) *ChunkDownload {
	pd.concurrency = concurrency
	return pd
}

func (pd *ChunkDownload) SetOutput(output io.Writer) *ChunkDownload {
	if output != nil {
		pd.output = output
	}
	return pd
}

func (pd *ChunkDownload) SetOutputFile(filename string) *ChunkDownload {
	pd.filename = filename
	return pd
}
func (pd *ChunkDownload) SetFileSize(totalBytes int64) *ChunkDownload {
	pd.totalBytes = totalBytes
	return pd
}

func (pd *ChunkDownload) SetOutputDirectory(outputDirectory string) *ChunkDownload {
	pd.outputDirectory = outputDirectory
	return pd
}

func getRangeTempFile(rangeStart, rangeEnd int64, workerDir string) string {
	return filepath.Join(workerDir, fmt.Sprintf("temp-%d-%d", rangeStart, rangeEnd))
}

func getRangeStartEnd(filename string) (int64, int64) {
	// 获取文件名部分，去掉路径
	baseName := filepath.Base(filename)

	// 去掉前缀 "temp-"
	parts := strings.Split(baseName, "-")
	if len(parts) != 3 || parts[0] != "temp" {
		return 0, 0 // 或者返回错误信息
	}

	// 解析 rangeStart 和 rangeEnd
	rangeStart, err1 := strconv.ParseInt(parts[1], 10, 64)
	rangeEnd, err2 := strconv.ParseInt(parts[2], 10, 64)

	if err1 != nil || err2 != nil {
		return 0, 0 // 或者返回错误信息
	}

	return rangeStart, rangeEnd
}

type downloadTask struct {
	index                int
	rangeStart, rangeEnd int64
	tempFilename         string
	tempFile             *os.File
}

func (pd *ChunkDownload) handleTask(t *downloadTask, ctx ...context.Context) {
	pd.wg.Add(1)
	defer pd.wg.Done()

	file, err := os.OpenFile(t.tempFilename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		pd.errCh <- err
		return
	}
	err = pd.client.Get(pd.url).
		SetHeader("Range", fmt.Sprintf("bytes=%d-%d", t.rangeStart, t.rangeEnd)).
		SetOutput(file).
		Do(ctx...).Err

	if err != nil {
		pd.errCh <- err
		return
	}
	t.tempFile = file
	pd.completeTask(t)
}

func (pd *ChunkDownload) startWorker(ctx ...context.Context) {
	for {
		select {
		case t := <-pd.taskCh:
			pd.handleTask(t, ctx...)
		case <-pd.doneCh:
			return
		}
	}
}

func (pd *ChunkDownload) mergeFile() {
	defer pd.wg.Done()
	file, err := pd.getOutputFile()
	if err != nil {
		pd.errCh <- err
		return
	}
	for i := 0; ; i++ {
		task := pd.popTask(i)
		tempFile, err := os.Open(task.tempFilename)
		if err != nil {
			pd.errCh <- err
			return
		}
		_, err = io.Copy(file, tempFile)
		tempFile.Close()
		if err != nil {
			pd.errCh <- err
			return
		}
		if i < pd.lastIndex {
			continue
		}
		break
	}

	err = os.RemoveAll(pd.tempDir)
	if err != nil {
		pd.errCh <- err
	}
}

func (pd *ChunkDownload) Do(ctx ...context.Context) error {
	err := pd.ensure()
	if err != nil {
		return err
	}
	for i := 0; i < pd.concurrency; i++ {
		go pd.startWorker(ctx...)
	}
	if pd.totalBytes == 0 {
		resp := pd.client.Head(pd.url).Do(ctx...)
		if resp.Err != nil {
			return resp.Err
		}
		if resp.ContentLength <= 0 {
			return fmt.Errorf("bad content length: %d", resp.ContentLength)
		}
		pd.totalBytes = resp.ContentLength
	}
	pd.lastIndex = int(math.Ceil(float64(pd.totalBytes)/float64(pd.chunkSize))) - 1
	pd.wg.Add(1)
	go pd.mergeFile()
	go func() {
		pd.wg.Wait()
		close(pd.wgDoneCh)
	}()
	err = pd.calTask()
	if err != nil {
		return err
	}

	select {
	case <-pd.wgDoneCh:
		close(pd.doneCh)
	case err := <-pd.errCh:
		return err
	}
	return nil
}

type Range struct {
	start int64
	end   int64
}

func (pd *ChunkDownload) calTask() error {
	totalBytes := pd.totalBytes
	start := int64(0)
	existRange, err := pd.existRange()
	if err != nil {
		return err
	}
	for i := 0; ; i++ {
		end := start + (pd.chunkSize - 1)
		if end > (totalBytes - 1) {
			end = totalBytes - 1
		}
		exist := false
		for _, r := range existRange {
			if !(end < r.start || start > r.end) {
				exist = true
				if start >= r.start && end <= r.end {
					// 完全重叠

				} else if start < r.start {
					// 左重叠
					end = r.start - 1
				} else if end > r.end {
					// 右重叠
					start = r.end + 1
				}
			}
		}

		if !exist {
			task := &downloadTask{
				tempFilename: getRangeTempFile(start, end, pd.tempDir),
				index:        i,
				rangeStart:   start,
				rangeEnd:     end,
			}
			pd.taskCh <- task
		}
		if end < (totalBytes - 1) {
			start = end + 1
			continue
		}
		break
	}
	//t.tempFilename = getRangeTempFile(t.rangeStart, t.rangeEnd, pd.tempDir)
	//// 先判断一下分块是不是存在，并且大小是需要下载的大小
	//existFile, err := IsExistFile(t.tempFilename)
	//if err != nil {
	//	pd.errCh <- err
	//	return
	//} else if existFile != nil {
	//	// 大小满足表示文件已经下载过，不必重复下载
	//	if existFile.Size() == t.rangeEnd-t.rangeStart+1 {
	//		pd.completeTask(t)
	//		return
	//	}
	//}
	return nil
}

func (pd *ChunkDownload) existRange() ([]Range, error) {
	dir, err := os.ReadDir(pd.tempDir)
	if err != nil {
		return nil, err
	}
	ranges := make([]Range, 0)
	for _, entry := range dir {
		name := entry.Name()
		s, e := getRangeStartEnd(name)
		if e-s > 0 {
			fileInfo, err := IsExistFile(pd.tempDir + "/" + name)
			if err != nil {
				return nil, err
			}
			if fileInfo != nil {
				// 文件大小不一致，执行清理
				if fileInfo.Size() != e-s+1 {
					_ = os.Remove(pd.tempDir + "/" + name)
				} else {
					ranges = append(ranges, Range{start: s, end: e})
				}
			}
		} else {
			_ = os.Remove(pd.tempDir + "/" + name)
		}
	}
	return ranges, nil
}

func (pd *ChunkDownload) getOutputFile() (io.Writer, error) {
	outputFile := pd.output
	if outputFile != nil {
		return outputFile, nil
	}
	if pd.filename == "" {
		u, err := urlpkg.Parse(pd.url)
		if err != nil {
			panic(err)
		}
		paths := strings.Split(u.Path, "/")
		for i := len(paths) - 1; i > 0; i-- {
			if paths[i] != "" {
				pd.filename = paths[i]
				break
			}
		}
		if pd.filename == "" {
			pd.filename = "download"
		}
	}
	if pd.outputDirectory != "" && !filepath.IsAbs(pd.filename) {
		err := os.MkdirAll(pd.outputDirectory, os.ModePerm)
		if err != nil {
			return nil, err
		}
		pd.filename = filepath.Join(pd.outputDirectory, pd.filename)
	}
	return os.OpenFile(pd.filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
}
