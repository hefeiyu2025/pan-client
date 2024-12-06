package internal

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
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

func GetWordPath() string {
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
