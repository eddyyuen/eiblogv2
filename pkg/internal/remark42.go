// Package internal provides ...
package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/eiblog/eiblog/pkg/config"
	"github.com/eiblog/eiblog/pkg/model"
	"io"
	"net/http"
	"net/url"
)

// disqus api
const (
	apiCommentsCount = "%s/api/v1/count?site=%s&%s"
)

// postsCountResp 评论数量响应
type postsRemark42CountResp struct {
	Count    int
	Response []struct {
		ID          string
		Posts       int
		Identifiers []string
	}
}

// PostRemark42Count 获取文章评论数量
func PostRemark42Count(articles map[string]*model.Article) error {
	if err := checkDisqusConfig(); err != nil {
		return err
	}

	vals := url.Values{}

	// batch get

	for _, article := range articles {
		vals.Set("url", "https://"+config.Conf.EiBlogApp.Host+"/post/"+article.Slug+".html")
		resp, err := httpGet(fmt.Sprintf(apiCommentsCount, config.Conf.EiBlogApp.Remark42.Domain, config.Conf.EiBlogApp.Remark42.SiteID, vals.Encode()))
		if err != nil {
			return err
		}

		b, err := io.ReadAll(resp.Body)
		if err != nil {
			_ = resp.Body.Close()
			return err
		}
		// check http status code
		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			return errors.New(string(b))
		}

		result := &postsRemark42CountResp{}
		err = json.Unmarshal(b, result)
		if err != nil {
			return err
		}

		article.Count = result.Count
		_ = resp.Body.Close()
	}
	return nil
}
