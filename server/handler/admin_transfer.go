package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"os"

	"github.com/ArtalkJS/ArtalkGo/internal/artransfer"
	"github.com/ArtalkJS/ArtalkGo/internal/query"
	"github.com/ArtalkJS/ArtalkGo/internal/utils"
	"github.com/ArtalkJS/ArtalkGo/server/common"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func AdminTransfer(router fiber.Router) {
	router.Post("/import", adminImport)
	router.Post("/import-upload", adminImportUpload)
	router.Post("/export", adminExport)
}

type ParamsAdminImport struct {
	Payload string `form:"payload"`
}

func adminImport(c *fiber.Ctx) error {
	var p ParamsAdminImport
	if isOK, resp := common.ParamsDecode(c, &p); !isOK {
		return resp
	}

	var payloadMapRaw map[string]interface{}
	err := json.Unmarshal([]byte(p.Payload), &payloadMapRaw)
	if err != nil {
		return common.RespError(c, "payload 解析错误", common.Map{
			"error": err,
		})
	}

	payloadMap := map[string]string{}
	for k, v := range payloadMapRaw {
		payloadMap[k] = utils.ToString(v) // convert all value to string
	}

	payloadArr := []string{}
	for k, v := range payloadMap {
		payloadArr = append(payloadArr, k+":"+v)
	}

	if !common.GetIsSuperAdmin(c) {
		user := common.GetUserByReq(c)
		if sitName, isExist := payloadMap["t_name"]; isExist {
			if !utils.ContainsStr(query.CookUser(&user).SiteNames, sitName) {
				return common.RespError(c, "禁止导入的目标站点名")
			}
		} else {
			return common.RespError(c, "请填写目标站点名")
		}
	}

	// TODO bcz 懒，先整这个缓冲输出，以后改成高级点的
	r := c.Response()

	r.Header.Add(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	r.SetStatusCode(fiber.StatusOK)

	buf := bytes.NewBufferString("")
	r.SetBodyStream(buf, -1)
	r.ImmediateHeaderFlush = true

	buf.Write([]byte(
		`<style>* { font-family: Menlo, Consolas, Monaco, monospace;word-wrap: break-word;white-space: pre-wrap;font-size: 13px; }</style>
		<script>function scroll() { if (!!document.body) { document.body.scrollTo(0, 999999999999); } }</script>`))

	artransfer.Assumeyes = true
	artransfer.HttpOutput = func(continueRun bool, text string) {
		buf.Write([]byte(text))
		buf.Write([]byte("<script>scroll();</script>"))
	}
	artransfer.RunImportArtrans(payloadArr)

	// 刷新 CORS 可信域名
	common.ReloadCorsAllowOrigins()

	return nil
}

func adminImportUpload(c *fiber.Ctx) error {
	// 获取 Form
	file, err := c.FormFile("file")
	if err != nil {
		logrus.Error(err)
		return common.RespError(c, "文件获取失败")
	}

	// 打开文件
	src, err := file.Open()
	if err != nil {
		logrus.Error(err)
		return common.RespError(c, "文件打开失败")
	}
	defer src.Close()

	// 读取文件
	buf, err := io.ReadAll(src)
	if err != nil {
		logrus.Error(err)
		return common.RespError(c, "文件读取失败")
	}

	tmpFile, err := os.CreateTemp("", "artalk-import-file-")
	if err != nil {
		logrus.Error(err)
		return common.RespError(c, "临时文件创建失败")
	}

	tmpFile.Write(buf)

	return common.RespData(c, common.Map{
		"filename": tmpFile.Name(),
	})
}

func adminExport(c *fiber.Ctx) error {
	jsonStr, err := artransfer.ExportArtransString(func(db *gorm.DB) *gorm.DB {
		if !common.GetIsSuperAdmin(c) {
			// 仅导出限定范围内的站点
			u := common.GetUserByReq(c)
			db = db.Where("site_name IN (?)", query.CookUser(&u).SiteNames)
		}

		return db
	})
	if err != nil {
		common.RespError(c, "导出错误", common.Map{
			"err": err,
		})
	}

	return common.RespData(c, common.Map{
		"data": jsonStr,
	})
}
