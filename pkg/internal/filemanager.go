// Package internal provides ...
package internal

import (
	"github.com/eiblog/eiblog/pkg/config"
	"io"
	"os"
	"path/filepath"
)

// params
type FileUploadParams struct {
	Name           string
	Size           int64
	Data           io.Reader
	NoCompletePath bool

	RootConf config.StaticFile
	Conf     config.LocalStor
}

type FileDeleteParams struct {
	Name           string
	Days           int
	NoCompletePath bool

	Conf config.LocalStor
}

// LocalUpload 上传文件
func LocalUpload(params FileUploadParams) (string, error) {
	key := params.Name
	if !params.NoCompletePath {
		key = filepath.Base(params.Name)
	}
	key = completeKey(key)

	localPath := params.Conf.LocalPath
	err := os.MkdirAll(localPath, 0644)
	if err != nil {
		return "", err
	}
	filePath := filepath.Join(localPath, key)
	err = os.MkdirAll(filepath.Dir(filePath), 0644)
	if err != nil {
		return "", err
	}
	file, err := os.Create(filePath)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(file, params.Data)
	if err != nil {
		return "", err
	}
	err = file.Close()
	if err != nil {
	}
	url := "https://" + params.RootConf.Domain + "/" + key
	return url, nil
}

// LocalDelete 删除文件
func LocalDelete(params FileDeleteParams) error {
	key := params.Name
	if !params.NoCompletePath {
		key = completeKey(params.Name)
	}

	localPath := params.Conf.LocalPath
	return os.Remove(filepath.Join(localPath, key))
}

// completeLocalKey 修复路径
func completeKey(name string) string {
	ext := filepath.Ext(name)

	switch ext {
	case ".bmp", ".png", ".jpg",
		".gif", ".ico", ".jpeg":

		name = "blog/img/" + name
	case ".mov", ".mp4":
		name = "blog/video/" + name
	case ".go", ".js", ".css",
		".cpp", ".php", ".rb",
		".java", ".py", ".sql",
		".lua", ".html", ".sh",
		".xml", ".cs":

		name = "blog/code/" + name
	case ".txt", ".md", ".ini",
		".yaml", ".yml", ".doc",
		".ppt", ".pdf":

		name = "blog/document/" + name
	case ".zip", ".rar", ".tar",
		".gz":

		name = "blog/archive/" + name
	default:
		name = "blog/other/" + name
	}
	return name
}
