// Package cache provides ...
package cache

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/eiblog/eiblog/pkg/cache/render"
	"github.com/eiblog/eiblog/pkg/cache/store"
	"github.com/eiblog/eiblog/pkg/config"
	"github.com/eiblog/eiblog/pkg/internal"
	"github.com/eiblog/eiblog/pkg/model"
	"github.com/eiblog/eiblog/tools"

	"github.com/sirupsen/logrus"
)

var (
	// Ei eiblog cache
	Ei *Cache

	// regenerate pages chan
	pagesCh     = make(chan string, 1)
	pageSeries  = "series-md"
	pageArchive = "archive-md"
)

func init() {
	store, err := store.NewStore(config.Conf.Database.Driver,
		config.Conf.Database.Source)
	if err != nil {
		panic(err)
	}
	// Ei init
	Ei = &Cache{
		Store:       store,
		TagArticles: make(map[string]model.SortedArticles),
		ArticlesMap: make(map[string]*model.Article),
	}
	err = Ei.loadAccount()
	if err != nil {
		panic(err)
	}
	// blogger
	err = Ei.loadBlogger()
	if err != nil {
		panic(err)
	}
	// articles
	err = Ei.loadArticles()
	if err != nil {
		panic(err)
	}
	go Ei.regeneratePages()
	go Ei.timerClean()
	go Ei.timerDisqus()
}

// Cache 整站缓存
type Cache struct {
	store.Store

	// load from db
	Blogger  *model.Blogger
	Account  *model.Account
	Articles model.SortedArticles

	// auto generate
	PageSeries   string // page
	Series       model.SortedSeries
	PageArchives string // page
	Archives     model.SortedArchives
	TagArticles  map[string]model.SortedArticles // tagname:articles
	ArticlesMap  map[string]*model.Article       // slug:article
}

// loadBlogger 博客信息
func (c *Cache) loadBlogger() error {
	blogapp := config.Conf.BlogApp
	blogger := &model.Blogger{
		BlogName:  blogapp.Blogger.BlogName,
		SubTitle:  blogapp.Blogger.SubTitle,
		BeiAn:     blogapp.Blogger.BeiAn,
		BTitle:    blogapp.Blogger.BTitle,
		Copyright: blogapp.Blogger.Copyright,
	}
	blogger, err := c.LoadOrCreateBlogger(blogger)
	if err != nil {
		return err
	}
	c.Blogger = blogger
	return nil
}

// loadAccount 账户账户信息
func (c *Cache) loadAccount() error {
	blogapp := config.Conf.BlogApp
	pwd := tools.EncryptPasswd(blogapp.Account.Password,
		blogapp.Account.Password)

	account := &model.Account{
		Username: blogapp.Account.Username,
		Password: pwd,
		Email:    blogapp.Account.Email,
		PhoneN:   blogapp.Account.PhoneNumber,
		Address:  blogapp.Account.Address,
	}
	account, err := c.LoadOrCreateAccount(account)
	if err != nil {
		return err
	}
	c.Account = account
	return nil
}

// loadArticles 文章信息
func (c *Cache) loadArticles() error {
	articles, err := c.LoadAllArticles()
	if err != nil {
		return err
	}
	sort.Sort(model.SortedArticles(articles))

	for i, v := range Ei.Articles {
		// 渲染页面
		render.GenerateExcerptMarkdown(v)

		c.ArticlesMap[v.Slug] = v
		// 分析文章
		if v.ID < config.Conf.BlogApp.General.StartID {
			continue
		}
		if i > 0 {
			v.Prev = Ei.Articles[i-1]
		}
		if Ei.Articles[i+1].ID >= config.Conf.BlogApp.General.StartID {
			v.Next = Ei.Articles[i+1]
		}
		c.rebuildArticle(v, false)
	}
	// 重建专题与归档
	pagesCh <- pageSeries
	pagesCh <- pageArchive
	return nil
}

// rebuildArticle 重建缓存tag、series、archive
func (c *Cache) rebuildArticle(article *model.Article, needSort bool) {
	// tag
	tags := strings.Split(article.Tags, ",")
	for _, tag := range tags {
		c.TagArticles[tag] = append(c.TagArticles[tag], article)
		if needSort {
			sort.Sort(c.TagArticles[tag])
		}
	}
	// series
	for i, series := range c.Series {
		if series.ID == article.SerieID {
			c.Series[i].Articles = append(c.Series[i].Articles, article)
			if needSort {
				sort.Sort(c.Series[i].Articles)
				pagesCh <- pageSeries // 重建专题
			}
		}
	}
	// archive
	y, m, _ := article.CreateTime.Date()
	for i, archive := range c.Archives {
		if ay, am, _ := archive.Time.Date(); y == ay && m == am {
			c.Archives[i].Articles = append(c.Archives[i].Articles, article)
		}
		if needSort {
			sort.Sort(c.Archives[i].Articles)
			pagesCh <- pageArchive // 重建归档
		}
		return
	}
	// 新建归档
	c.Archives = append(c.Archives, &model.Archive{
		Time:     article.CreateTime,
		Articles: model.SortedArticles{article},
	})
	if needSort { // 重建归档
		pagesCh <- pageArchive
	}
}

// regeneratePages 重新生成series,archive页面
func (c *Cache) regeneratePages() {
	for {
		switch page := <-pagesCh; page {
		case pageSeries:
			sort.Sort(c.Series)
			buf := bytes.Buffer{}
			buf.WriteString(c.Blogger.SeriesSay)
			buf.WriteString("\n\n")
			for _, series := range c.Series {
				buf.WriteString(fmt.Sprintf("### %s{#toc-%d}", series.Name, series.ID))
				buf.WriteByte('\n')
				buf.WriteString(series.Desc)
				buf.WriteString("\n\n")
				for _, article := range series.Articles {
					//eg. * [标题一](/post/hello-world.html) <span class="date">(Man 02, 2006)</span>
					str := fmt.Sprintf(`* [%s](/post/%s.html) <span class="date">(%s)</span>`,
						article.Title, article.Slug, article.CreateTime.Format("Jan 02, 2006"))
					buf.WriteString(str)
				}
				buf.WriteString("\n\n")
			}
			c.PageSeries = string(render.RenderPage(buf.Bytes()))
		case pageArchive:
			sort.Sort(c.Archives)
			buf := bytes.Buffer{}
			buf.WriteString(c.Blogger.ArchivesSay + "\n")
			var (
				currentYear string
				gt12Month   = len(Ei.Archives) > 12
			)
			for _, archive := range c.Archives {
				if gt12Month {
					year := archive.Time.Format("2006 年")
					if currentYear != year {
						currentYear = year
						buf.WriteString(fmt.Sprintf("\n### %s\n\n", archive.Time.Format("2006 年")))
					}
				} else {
					buf.WriteString(fmt.Sprintf("\n### %s\n\n", archive.Time.Format("2006年1月")))
				}
				for i, article := range archive.Articles {
					if i == 0 && gt12Month {
						str := fmt.Sprintf(`* *[%s](/post/%s.html) <span class="date">(%s)</span>`,
							article.Title, article.Slug, article.CreateTime.Format("Jan 02, 2006"))
						buf.WriteString(str)
					} else {
						str := fmt.Sprintf(`* [%s](/post/%s.html) <span class="date">(%s)</span>`,
							article.Title, article.Slug, article.CreateTime.Format("Jan 02, 2006"))
						buf.WriteString(str)
					}
					buf.WriteByte('\n')
				}
			}
			c.PageArchives = string(render.RenderPage(buf.Bytes()))
		}
	}
}

// timerClean 定时清理文章
func (c *Cache) timerClean() {
	dur := time.Duration(config.Conf.BlogApp.General.Clean)
	ticker := time.NewTicker(dur * time.Hour)

	for range ticker.C {
		err := c.CleanArticles()
		if err != nil {
			logrus.Error("cache.timerClean.CleanArticles: ", err)
		}
	}
}

// timerDisqus disqus定时操作
func (c *Cache) timerDisqus() {
	dur := time.Duration(config.Conf.BlogApp.Disqus.Interval)
	ticker := time.NewTicker(dur * time.Hour)

	for range ticker.C {
		err := internal.PostsCount(c.ArticlesMap)
		if err != nil {
			logrus.Error("cache.timerDisqus.PostsCount: ", err)
		}
	}
}

// PageArticles 文章翻页
func (c *Cache) PageArticles(page int, pageSize int) (prev,
	next int, articles []*model.Article) {

	var l int
	for l = len(c.Articles); l > 0; l-- {
		if c.Articles[l-1].ID >= config.Conf.BlogApp.General.StartID {
			break
		}
	}
	if l == 0 {
		return 0, 0, nil
	}
	m := l / pageSize
	if d := l % pageSize; d > 0 {
		m++
	}
	if page > m {
		page = m
	}
	if page > 1 {
		prev = page - 1
	}
	if page < m {
		next = page + 1
	}
	s := (page - 1) * pageSize
	e := page * pageSize
	if e > l {
		e = l
	}
	articles = c.Articles[s:e]
	return
}
