package internal

import (
	"crypto/md5"
	"encoding/hex"
	logger "github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
	"time"
)

// 文件处理

// IsEmptyDir 判断目录是否为空
func IsEmptyDir(dirPath string) (bool, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return false, err
	}
	defer dir.Close()
	//如果目录不为空，Readdirnames 会返回至少一个文件名
	_, err = dir.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

func GetWorkPath() string {
	workDir, _ := os.Getwd()
	return workDir
}
func GetProcessPath() string {
	// 添加运行目录
	process, _ := os.Executable()
	return filepath.Dir(process)
}

func IsExistFile(filePath string) (os.FileInfo, error) {
	stat, err := os.Stat(filePath)
	if err == nil {
		return stat, nil
	}
	if os.IsNotExist(err) {
		return nil, nil
	}
	return nil, err
}

// MD5

func Md5HashStr(str string) string {
	// 计算JSON数据的MD5
	hash := md5.Sum([]byte(str))
	return hex.EncodeToString(hash[:])
}

// SHA1

// Log

func LogProgress(prefix, fileName string, startTime time.Time, thisOperated, operated, totalSize int64, mustLog bool) {
	elapsed := time.Since(startTime).Seconds()
	var speed float64
	if elapsed == 0 {
		speed = float64(thisOperated) / 1024
	} else {
		speed = float64(thisOperated) / 1024 / elapsed // KB/s
	}

	// 计算进度百分比
	percent := float64(operated) / float64(totalSize) * 100
	if Config.Server.Debug || mustLog {
		logger.Infof("\r %s %s: %.2f%% (%d/%d bytes, %.2f KB/s)", prefix, fileName, percent, operated, totalSize, speed)
	}
	if operated == totalSize {
		logger.Infof("%s %s: %.2f%% (%d/%d bytes, %.2f KB/s), cost %.2f s", prefix, fileName, percent, operated, totalSize, speed, elapsed)
	}
}
