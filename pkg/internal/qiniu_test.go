// Package internal provides ...
package internal

import (
	"os"
	"testing"
	"time"

	"github.com/eiblog/eiblog/pkg/config"
)

func TestQiniuUpload(t *testing.T) {

	f, _ := os.Open("qiniu_test.go")
	fi, _ := f.Stat()

	type args struct {
		params QiniuUploadParams
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{"1", args{params: QiniuUploadParams{
			Name: "test-" + time.Now().Format("200601021504059999") + ".go",
			Size: fi.Size(),
			Data: f,
			RootConf: config.StaticFile{
				Type:   "qiniu",
				Domain: "bu.st.deepzz.com",
			},
			Conf: config.Qiniu{
				AccessKey: os.Getenv("QINIU_ACCESSKEY"),
				SecretKey: os.Getenv("QINIU_SECRETKEY"),
				Bucket:    os.Getenv("QINIU_BUCKET"),
			},
		}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := QiniuUpload(tt.args.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("QiniuUpload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			t.Logf("QiniuUpload() = %v", got)
		})
	}
}

func TestQiniuContent(t *testing.T) {
	params := QiniuContentParams{
		RootConf: config.StaticFile{
			Type:   "qiniu",
			Domain: "bu.st.deepzz.com",
		},
		Conf: config.Qiniu{
			AccessKey: os.Getenv("QINIU_ACCESSKEY"),
			SecretKey: os.Getenv("QINIU_SECRETKEY"),
			Bucket:    os.Getenv("QINIU_BUCKET"),
		},
	}
	_, err := QiniuContent(params)
	if err != nil {
		t.Errorf("QiniuList error = %v", err)
	}
}
