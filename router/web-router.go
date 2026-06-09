package router

import (
	"embed"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/webseo"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

// ThemeAssets holds the embedded frontend assets for both themes.
type ThemeAssets struct {
	DefaultBuildFS   embed.FS
	DefaultIndexPage []byte
	ClassicBuildFS   embed.FS
	ClassicIndexPage []byte
}

func SetWebRouter(router *gin.Engine, assets ThemeAssets) {
	defaultFS := common.EmbedFolder(assets.DefaultBuildFS, "web/default/dist")
	classicFS := common.EmbedFolder(assets.ClassicBuildFS, "web/classic/dist")
	themeFS := common.NewThemeAwareFS(defaultFS, classicFS)

	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(middleware.GlobalWebRateLimit())
	router.Use(middleware.Cache())
	router.GET("/robots.txt", func(c *gin.Context) {
		c.Set(middleware.RouteTagKey, "web")
		c.Header("Cache-Control", "public, max-age=3600")
		c.String(http.StatusOK, webseo.BuildRobotsTxt(system_setting.ServerAddress))
	})
	router.GET("/sitemap.xml", func(c *gin.Context) {
		c.Set(middleware.RouteTagKey, "web")
		c.Header("Cache-Control", "public, max-age=3600")
		c.Data(http.StatusOK, "application/xml; charset=utf-8", []byte(webseo.BuildSitemapXMLForTheme(system_setting.ServerAddress, model.GetPricing(), common.GetTheme())))
	})
	router.Use(static.Serve("/", themeFS))
	router.NoRoute(func(c *gin.Context) {
		c.Set(middleware.RouteTagKey, "web")
		if strings.HasPrefix(c.Request.RequestURI, "/v1") || strings.HasPrefix(c.Request.RequestURI, "/api") || strings.HasPrefix(c.Request.RequestURI, "/assets") {
			controller.RelayNotFound(c)
			return
		}
		c.Header("Cache-Control", "no-cache")
		theme := common.GetTheme()
		pricings := model.GetPricing()
		meta := webseo.ResolveMetaForTheme(c.Request.RequestURI, system_setting.ServerAddress, pricings, theme)
		body := webseo.BuildBodyContent(meta, c.Request.RequestURI, system_setting.ServerAddress, pricings, theme)
		if theme == "classic" {
			c.Data(http.StatusOK, "text/html; charset=utf-8", webseo.RenderIndexHTML(assets.ClassicIndexPage, meta, body))
		} else {
			c.Data(http.StatusOK, "text/html; charset=utf-8", webseo.RenderIndexHTML(assets.DefaultIndexPage, meta, body))
		}
	})
}
