// Package main provides ...
package main

import (
	"flag"
	"fmt"
	"github.com/eiblog/eiblog/pkg/config"
	"github.com/eiblog/eiblog/pkg/core/eiblog"
	"github.com/eiblog/eiblog/pkg/core/eiblog/admin"
	"github.com/eiblog/eiblog/pkg/core/eiblog/file"
	"github.com/eiblog/eiblog/pkg/core/eiblog/page"
	"github.com/eiblog/eiblog/pkg/core/eiblog/swag"
	"github.com/eiblog/eiblog/pkg/mid"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/zh-five/xdaemon"
	"path/filepath"
)

func main() {
	d := flag.Bool("d", false, "run in background")
	logFile := flag.String("l", "eiblog.log", "log path")
	flag.Parse()

	if *d {
		_, err := xdaemon.Background(*logFile, true)
		if err != nil {
			fmt.Println(err)
		}
		return
	}

	logrus.Info("EiBlog start, app name " + config.Conf.EiBlogApp.Name)

	endRun := make(chan error, 1)

	runHTTPServer(endRun)
	fmt.Println(<-endRun)
}

func runHTTPServer(endRun chan error) {
	if !config.Conf.EiBlogApp.EnableHTTP {
		return
	}

	if config.Conf.RunMode == config.ModeProd {
		gin.SetMode(gin.ReleaseMode)
	}
	e := gin.Default()
	// middleware
	e.Use(mid.UserMiddleware())
	e.Use(mid.SessionMiddleware(mid.SessionOpts{
		Name:   "su",
		Secure: config.Conf.RunMode == config.ModeProd,
		Secret: []byte("ZGlzvcmUoMTAsICI="),
	}))

	// swag
	swag.RegisterRoutes(e)

	// static files, page
	root := filepath.Join(config.WorkDir, "assets")
	e.Static("/static", root)

	// static files
	file.RegisterRoutes(e)
	// frontend pages
	page.RegisterRoutes(e)
	// unauthz api
	admin.RegisterRoutes(e)

	// admin router
	group := e.Group("/admin", eiblog.AuthFilter)
	{
		page.RegisterRoutesAuthz(group)
		admin.RegisterRoutesAuthz(group)
	}

	// start
	address := fmt.Sprintf(":%d", config.Conf.EiBlogApp.HTTPPort)
	go func() {
		endRun <- e.Run(address)
	}()
	fmt.Println("HTTP server running on: " + address)
}
