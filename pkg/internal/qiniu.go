// Package internal provides ...
package internal

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/eiblog/eiblog/pkg/config"

	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"
)

// QiniuUploadParams upload params
type QiniuUploadParams struct {
	Name           string
	Size           int64
	Data           io.Reader
	NoCompletePath bool

	RootConf config.StaticFile
	Conf     config.Qiniu
}

// QiniuUpload 上传文件
func QiniuUpload(params QiniuUploadParams) (string, error) {
	if params.Conf.AccessKey == "" ||
		params.Conf.SecretKey == "" {
		return "", errors.New("qiniu config error")
	}
	key := params.Name
	if !params.NoCompletePath {
		key = filepath.Base(params.Name)
	}

	mac := qbox.NewMac(params.Conf.AccessKey,
		params.Conf.SecretKey)
	// 设置上传策略
	putPolicy := &storage.PutPolicy{
		Scope:      params.Conf.Bucket,
		Expires:    3600,
		InsertOnly: 1,
	}
	// 上传token
	uploadToken := putPolicy.UploadToken(mac)
	// 上传配置
	region, err := storage.GetRegion(params.Conf.AccessKey, params.Conf.Bucket)
	if err != nil {
		return "", err
	}
	cfg := &storage.Config{
		UseHTTPS: true,
		Region:   region,
	}
	// uploader
	uploader := storage.NewFormUploader(cfg)
	ret := new(storage.PutRet)
	putExtra := &storage.PutExtra{}

	err = uploader.Put(context.Background(), ret, uploadToken,
		key, params.Data, params.Size, putExtra)
	if err != nil {
		return "", err
	}
	url := "https://" + params.RootConf.Domain + "/" + key
	return url, nil
}

// DeleteParams delete params
type QiniuDeleteParams struct {
	Name           string
	Days           int
	NoCompletePath bool

	Conf config.Qiniu
}

// QiniuDelete 删除文件
func QiniuDelete(params QiniuDeleteParams) error {
	key := params.Name
	if !params.NoCompletePath {
		key = completeQiniuKey(params.Name)
	}

	mac := qbox.NewMac(params.Conf.AccessKey,
		params.Conf.SecretKey)
	// 上传配置
	region, err := storage.GetRegion(params.Conf.AccessKey, params.Conf.Bucket)
	if err != nil {
		return err
	}
	cfg := &storage.Config{
		UseHTTPS: true,
		Region:   region,
	}
	// manager
	bucketManager := storage.NewBucketManager(mac, cfg)
	// Delete
	if params.Days > 0 {
		return bucketManager.DeleteAfterDays(params.Conf.Bucket, key, params.Days)
	}
	return bucketManager.Delete(params.Conf.Bucket, key)
}

// ContentParams list params
type QiniuContentParams struct {
	Prefix string

	RootConf config.StaticFile
	Conf     config.Qiniu
}

// QiniuContent 获取文件列表
func QiniuContent(params QiniuContentParams) ([]byte, error) {
	mac := qbox.NewMac(params.Conf.AccessKey,
		params.Conf.SecretKey)
	// region
	region, err := storage.GetRegion(params.Conf.AccessKey, params.Conf.Bucket)
	if err != nil {
		return nil, err
	}
	cfg := &storage.Config{
		UseHTTPS: true,
		Region:   region,
	}
	// manager
	bucketManager := storage.NewBucketManager(mac, cfg)
	// list file
	files, _, _, _, err := bucketManager.ListFiles(params.Conf.Bucket, params.Prefix, "", "", 1)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, errors.New("no file")
	}
	deadline := time.Now().Add(time.Second * 60).Unix()
	url := storage.MakePrivateURLv2(mac, "https://"+params.RootConf.Domain, files[0].Key, deadline)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	return io.ReadAll(resp.Body)
}

// completeQiniuKey 修复路径
func completeQiniuKey(name string) string {
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
