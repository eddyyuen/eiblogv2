// Package page provides ...
package page

import (
	"bytes"
	"context"
	"fmt"
	htemplate "html/template"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/eiblog/eiblog/pkg/cache"
	"github.com/eiblog/eiblog/pkg/config"
	"github.com/eiblog/eiblog/pkg/internal"
	"github.com/eiblog/eiblog/tools"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// baseFEParams 基础参数
func baseFEParams(c *gin.Context) gin.H {
	version := config.Conf.EiBlogApp.StaticVersion

	return gin.H{
		"BlogName":   cache.Ei.Blogger.BlogName,
		"SubTitle":   cache.Ei.Blogger.SubTitle,
		"BTitle":     cache.Ei.Blogger.BTitle,
		"BeiAn":      cache.Ei.Blogger.BeiAn,
		"Domain":     config.Conf.EiBlogApp.Host,
		"CopyYear":   time.Now().Year(),
		"Twitter":    config.Conf.EiBlogApp.Twitter,
		"StaticFile": config.Conf.EiBlogApp.StaticFile,
		"Disqus":     config.Conf.EiBlogApp.Disqus,
		"AdSense":    config.Conf.EiBlogApp.Google.AdSense,
		"Version":    version,
	}
}

// handleNotFound not found page
func handleNotFound(c *gin.Context) {
	params := baseFEParams(c)
	params["Title"] = "Not Found"
	params["Description"] = "404 Not Found"
	params["Path"] = ""
	c.Status(http.StatusNotFound)
	renderHTMLHomeLayout(c, "notfound", params)
}

// handleHomePage 首页
func handleHomePage(c *gin.Context) {
	params := baseFEParams(c)
	params["Title"] = cache.Ei.Blogger.BTitle + " | " + cache.Ei.Blogger.SubTitle
	params["Description"] = "博客首页，" + cache.Ei.Blogger.SubTitle
	params["Path"] = c.Request.URL.Path
	params["CurrentPage"] = "blog-home"
	pn, err := strconv.Atoi(c.Query("pn"))
	if err != nil || pn < 1 {
		pn = 1
	}
	params["Prev"], params["Next"], params["List"] = cache.Ei.PageArticleFE(pn,
		config.Conf.EiBlogApp.General.PageNum)

	renderHTMLHomeLayout(c, "home", params)
}

// handleArticlePage 文章页
func handleArticlePage(c *gin.Context) {
	slug := c.Param("slug")
	if !strings.HasSuffix(slug, ".html") || cache.Ei.ArticlesMap[slug[:len(slug)-5]] == nil {
		handleNotFound(c)
		return
	}
	article := cache.Ei.ArticlesMap[slug[:len(slug)-5]]
	params := baseFEParams(c)
	params["Title"] = article.Title + " | " + cache.Ei.Blogger.BTitle
	params["Path"] = c.Request.URL.Path
	params["CurrentPage"] = "post-" + article.Slug
	params["Article"] = article

	var name string
	switch slug {
	case "blogroll.html":
		name = "blogroll"
		params["Description"] = "友情连接，" + cache.Ei.Blogger.SubTitle
	case "about.html":
		name = "about"
		params["Description"] = "关于作者，" + cache.Ei.Blogger.SubTitle
	default:
		params["Description"] = article.Desc + "，" + cache.Ei.Blogger.SubTitle
		name = "article"
		params["Copyright"] = cache.Ei.Blogger.Copyright
		if !article.UpdatedAt.IsZero() {
			params["Days"] = int(time.Now().Sub(article.UpdatedAt).Hours()) / 24
		} else {
			params["Days"] = int(time.Now().Sub(article.CreatedAt).Hours()) / 24
		}
		if article.SerieID > 0 {
			for _, series := range cache.Ei.Series {
				if series.ID == article.SerieID {
					params["Serie"] = series
				}
			}
		}
	}
	renderHTMLHomeLayout(c, name, params)
}

// handleSeriesPage 专题页
func handleSeriesPage(c *gin.Context) {
	params := baseFEParams(c)
	params["Title"] = "专题 | " + cache.Ei.Blogger.BTitle
	params["Description"] = "专题列表，" + cache.Ei.Blogger.SubTitle
	params["Path"] = c.Request.URL.Path
	params["CurrentPage"] = "series"
	params["Article"] = cache.Ei.PageSeries
	renderHTMLHomeLayout(c, "series", params)
}

// handleArchivePage 归档页
func handleArchivePage(c *gin.Context) {
	params := baseFEParams(c)
	params["Title"] = "归档 | " + cache.Ei.Blogger.BTitle
	params["Description"] = "博客归档，" + cache.Ei.Blogger.SubTitle
	params["Path"] = c.Request.URL.Path
	params["CurrentPage"] = "archives"
	params["Article"] = cache.Ei.PageArchives
	renderHTMLHomeLayout(c, "archives", params)
}

// handleSearchPage 搜索页
func handleSearchPage(c *gin.Context) {
	params := baseFEParams(c)
	params["Title"] = "站内搜索 | " + cache.Ei.Blogger.BTitle
	params["Description"] = "站内搜索，" + cache.Ei.Blogger.SubTitle
	params["Path"] = ""
	params["CurrentPage"] = "search-post"

	q := strings.TrimSpace(c.Query("q"))
	if q != "" {
		start, err := strconv.Atoi(c.Query("start"))
		if start < 1 || err != nil {
			start = 1
		}
		params["Word"] = q

		vals := c.Request.URL.Query()
		result, err := internal.ElasticSearch(q, config.Conf.EiBlogApp.General.PageNum, start-1)
		if err != nil {
			logrus.Error("HandleSearchPage.ElasticSearch: ", err)
		} else {
			result.Took /= 1000
			for i, v := range result.Hits.Hits {
				article := cache.Ei.ArticlesMap[v.Source.Slug]
				if len(v.Highlight.Content) == 0 && article != nil {
					result.Hits.Hits[i].Highlight.Content = []string{article.Excerpt}
				}
			}
			params["SearchResult"] = result
			if num := start - config.Conf.EiBlogApp.General.PageNum; num > 0 {
				vals.Set("start", fmt.Sprint(num))
				params["Prev"] = vals.Encode()
			}
			if num := start + config.Conf.EiBlogApp.General.PageNum; result.Hits.Total >= num {
				vals.Set("start", fmt.Sprint(num))
				params["Next"] = vals.Encode()
			}
		}
	} else {
		params["HotWords"] = config.Conf.EiBlogApp.HotWords
	}
	renderHTMLHomeLayout(c, "search", params)
}

// disqusComments 服务端获取评论详细
type disqusComments struct {
	ErrNo  int    `json:"errno"`
	ErrMsg string `json:"errmsg"`
	Data   struct {
		Next     string           `json:"next"`
		Total    int              `json:"total"`
		Comments []commentsDetail `json:"comments"`
		Thread   string           `json:"thread"`
	} `json:"data"`
}

// handleDisqusList 获取评论列表
func handleDisqusList(c *gin.Context) {
	dcs := &disqusComments{}
	defer c.JSON(http.StatusOK, dcs)

	slug := c.Param("slug")
	cursor := c.Query("cursor")
	artc := cache.Ei.ArticlesMap[slug]
	if artc != nil {
		dcs.Data.Thread = artc.Thread
	}
	postsList, err := internal.PostsList(artc, cursor)
	if err != nil {
		logrus.Error("hadnleDisqusList.PostsList: ", err)
		dcs.ErrNo = 0
		dcs.ErrMsg = "系统错误"
		return
	}
	dcs.ErrNo = postsList.Code
	if postsList.Cursor.HasNext {
		dcs.Data.Next = postsList.Cursor.Next
	}
	dcs.Data.Total = len(postsList.Response)
	dcs.Data.Comments = make([]commentsDetail, len(postsList.Response))
	for i, v := range postsList.Response {
		if dcs.Data.Thread == "" {
			dcs.Data.Thread = v.Thread
		}
		dcs.Data.Comments[i] = commentsDetail{
			ID:           v.ID,
			Name:         v.Author.Name,
			Parent:       v.Parent,
			URL:          v.Author.ProfileURL,
			Avatar:       v.Author.Avatar.Cache,
			CreatedAtStr: tools.ConvertStr(v.CreatedAt),
			Message:      v.Message,
			IsDeleted:    v.IsDeleted,
		}
	}
	// query thread & update
	if artc != nil && artc.Thread == "" {
		if dcs.Data.Thread != "" {
			artc.Thread = dcs.Data.Thread
		} else if internal.ThreadDetails(artc) == nil {
			dcs.Data.Thread = artc.Thread
		}
		cache.Ei.UpdateArticle(context.Background(), artc.ID,
			map[string]interface{}{
				"thread": artc.Thread,
			})
	}
}

// handleDisqusPage 评论页
func handleDisqusPage(c *gin.Context) {
	array := strings.Split(c.Param("slug"), "|")
	if len(array) != 4 || array[1] == "" {
		c.String(http.StatusOK, "出错啦。。。")
		return
	}
	article := cache.Ei.ArticlesMap[array[0]]
	params := gin.H{
		"Title":  "发表评论 | " + cache.Ei.Blogger.BTitle,
		"ATitle": article.Title,
		"Thread": array[1],
		"Slug":   article.Slug,
	}
	renderHTMLHomeLayout(c, "disqus.html", params)
}

// 发表评论
// [thread:[5279901489] parent:[] identifier:[post-troubleshooting-https]
// next:[] author_name:[你好] author_email:[chenqijing2@163.com] message:[fdsfdsf]]
type disqusCreate struct {
	ErrNo  int            `json:"errno"`
	ErrMsg string         `json:"errmsg"`
	Data   commentsDetail `json:"data"`
}

type commentsDetail struct {
	ID           string `json:"id"`
	Parent       int    `json:"parent"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	Avatar       string `json:"avatar"`
	CreatedAtStr string `json:"createdAtStr"`
	Message      string `json:"message"`
	IsDeleted    bool   `json:"isDeleted"`
}

// handleDisqusCreate 评论文章
func handleDisqusCreate(c *gin.Context) {
	resp := &disqusCreate{}
	defer c.JSON(http.StatusOK, resp)

	msg := c.PostForm("message")
	email := c.PostForm("author_email")
	name := c.PostForm("author_name")
	thread := c.PostForm("thread")
	identifier := c.PostForm("identifier")
	if msg == "" || email == "" || name == "" || thread == "" || identifier == "" {
		resp.ErrNo = 1
		resp.ErrMsg = "参数错误"
		return
	}
	logrus.Infof("email: %s comments: %s", email, thread)

	comment := internal.PostComment{
		Message:     msg,
		Parent:      c.PostForm("parent"),
		Thread:      thread,
		AuthorEmail: email,
		AuthorName:  name,
		Identifier:  identifier,
		IPAddress:   c.ClientIP(),
	}
	postDetail, err := internal.PostCreate(&comment)
	if err != nil {
		logrus.Error("handleDisqusCreate.PostCreate: ", err)
		resp.ErrNo = 1
		resp.ErrMsg = "提交评论失败，请重试"
		return
	}
	err = internal.PostApprove(postDetail.Response.ID)
	if err != nil {
		logrus.Error("handleDisqusCreate.PostApprove: ", err)
		resp.ErrNo = 1
		resp.ErrMsg = "提交评论失败，请重试"
	}
	resp.ErrNo = 0
	resp.Data = commentsDetail{
		ID:           postDetail.Response.ID,
		Name:         name,
		Parent:       postDetail.Response.Parent,
		URL:          postDetail.Response.Author.ProfileURL,
		Avatar:       postDetail.Response.Author.Avatar.Cache,
		CreatedAtStr: tools.ConvertStr(postDetail.Response.CreatedAt),
		Message:      postDetail.Response.Message,
		IsDeleted:    postDetail.Response.IsDeleted,
	}
}

// handleBeaconPage 服务端推送谷歌统计
// https://www.thyngster.com/ga4-measurement-protocol-cheatsheet/
func handleBeaconPage(c *gin.Context) {
	ua := c.Request.UserAgent()

	vals := c.Request.URL.Query()
	vals.Set("v", config.Conf.EiBlogApp.Google.V)
	vals.Set("tid", config.Conf.EiBlogApp.Google.Tid)
	cookie, _ := c.Cookie("u")
	vals.Set("cid", cookie)

	vals.Set("dl", c.Request.Referer())                        // document location
	vals.Set("en", "page_view")                                // event name
	vals.Set("sct", "1")                                       // Session Count
	vals.Set("seg", "1")                                       // Session Engagment
	vals.Set("_uip", c.ClientIP())                             // user ip
	vals.Set("_p", fmt.Sprint(201226219+rand.Intn(499999999))) // random page load hash
	vals.Set("_ee", "1")                                       // external event
	go func() {
		url := config.Conf.EiBlogApp.Google.URL + "?" + vals.Encode()
		req, err := http.NewRequest("POST", url, nil)
		if err != nil {
			logrus.Error("HandleBeaconPage.NewRequest: ", err)
			return
		}
		req.Header.Set("User-Agent", ua)
		req.Header.Set("Sec-Ch-Ua", c.GetHeader("Sec-Ch-Ua"))
		req.Header.Set("Sec-Ch-Ua-Platform", c.GetHeader("Sec-Ch-Ua-Platform"))
		req.Header.Set("Sec-Ch-Ua-Mobile", c.GetHeader("Sec-Ch-Ua-Mobile"))
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			logrus.Error("HandleBeaconPage.Do: ", err)
			return
		}
		defer res.Body.Close()
		data, err := ioutil.ReadAll(res.Body)
		if err != nil {
			logrus.Error("HandleBeaconPage.ReadAll: ", err)
			return
		}
		if res.StatusCode/100 != 2 {
			logrus.Error(string(data))
		}
	}()
	c.Status(http.StatusNoContent)
}

// renderHTMLHomeLayout homelayout html
func renderHTMLHomeLayout(c *gin.Context, name string, data gin.H) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	// special page
	if name == "disqus.html" {
		err := htmlTmpl.ExecuteTemplate(c.Writer, name, data)
		if err != nil {
			panic(err)
		}
		return
	}
	buf := bytes.Buffer{}
	err := htmlTmpl.ExecuteTemplate(&buf, name, data)
	if err != nil {
		panic(err)
	}
	data["LayoutContent"] = htemplate.HTML(buf.String())
	err = htmlTmpl.ExecuteTemplate(c.Writer, "homeLayout.html", data)
	if err != nil {
		panic(err)
	}
	if c.Writer.Status() == 0 {
		c.Status(http.StatusOK)
	}
}
