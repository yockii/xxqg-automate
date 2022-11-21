package study

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"gitee.com/chunanyong/zorm"
	"github.com/google/uuid"
	"github.com/panjf2000/ants/v2"
	logger "github.com/sirupsen/logrus"
	"github.com/yockii/qscore/pkg/config"

	"xxqg-automate/internal/model"
	"xxqg-automate/internal/service"
	"xxqg-automate/internal/util"
)

func init() {
	loadLoginJobs()
}

func loadLoginJobs() {
	jobs, err := service.JobService.FindList(context.Background(),
		zorm.NewSelectFinder(model.JobTableName).Append("WHERE status=2"),
		nil,
	)
	if err != nil {
		logger.Errorln(err)
		return
	}
	for _, job := range jobs {
		ants.Submit(func() {
			checkLogin(job)
			service.JobService.DeleteById(context.Background(), job.Id)
		})
	}
}

type checkQrCodeResp struct {
	Code    string `json:"code"`
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

func GetXxqgRedirectUrl(dingIds ...string) (ru string, err error) {
	dingId := ""
	if len(dingIds) > 0 {
		dingId = dingIds[0]
	}
	client := util.GetClient()
	type generateResp struct {
		Success   bool        `json:"success"`
		ErrorCode interface{} `json:"errorCode"`
		ErrorMsg  interface{} `json:"errorMsg"`
		Result    string      `json:"result"`
		Arguments interface{} `json:"arguments"`
	}
	g := new(generateResp)
	_, err = client.R().SetResult(g).Get("https://login.xuexi.cn/user/qrcode/generate")
	if err != nil {
		logger.Errorln(err.Error())
		return
	}
	logger.Infoln(g.Result)
	codeURL := fmt.Sprintf("https://login.xuexi.cn/login/qrcommit?showmenu=false&code=%v&appId=dingoankubyrfkttorhpou", g.Result)

	code := g.Result
	ants.Submit(func() {
		job := &model.Job{
			Status: 2,
			Code:   code + "|" + dingId,
		}
		service.JobService.Save(context.Background(), job)

		checkLogin(job)

		service.JobService.DeleteById(context.Background(), job.Id)
	})
	ru = config.GetString("xxqg.schema") + url.QueryEscape(codeURL)
	return
}

func checkLogin(job *model.Job) {
	client := util.GetClient()
	codeWithDingId := strings.Split(job.Code, "|")
	code := codeWithDingId[0]
	dingId := codeWithDingId[1]
	for i := 0; i < 150; i++ {
		res := new(checkQrCodeResp)
		_, err := client.R().SetResult(res).SetFormData(map[string]string{
			"qrCode":   code,
			"goto":     "https://oa.xuexi.cn",
			"pdmToken": "",
		}).SetHeader("content-type", "application/x-www-form-urlencoded;charset=UTF-8").
			Post("https://login.xuexi.cn/login/login_with_qr")
		if err != nil {
			logger.Error(err)
			continue
		}
		if !res.Success {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		type signResp struct {
			Data struct {
				Sign string `json:"sign"`
			} `json:"data"`
			Message string      `json:"message"`
			Code    int         `json:"code"`
			Error   interface{} `json:"error"`
			Ok      bool        `json:"ok"`
		}
		s := res.Data
		sign := new(signResp)
		_, err = client.R().SetResult(sign).Get("https://pc-api.xuexi.cn/open/api/sns/sign")
		if err != nil {
			logger.Errorln(err)
			return
		}
		s2 := strings.Split(s, "=")[1]
		response, err := client.R().SetQueryParams(map[string]string{
			"code":  s2,
			"state": sign.Data.Sign + uuid.New().String(),
		}).Get("https://pc-api.xuexi.cn/login/secure_check")
		if err != nil {
			logger.Errorln(err)
			return
		}
		user, err := GetUserInfo(response.Cookies())
		if err != nil {
			logger.Errorln(err)
			return
		}
		// 登录成功
		user.Token = response.Cookies()[0].Value
		user.LoginTime = time.Now().Unix()
		user.DingtalkId = dingId
		service.UserService.UpdateByUid(context.Background(), user)
		return
	}
}
