package study

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/panjf2000/ants/v2"
	"github.com/sirupsen/logrus"
	"github.com/yockii/qscore/pkg/config"

	"xxqg-automate/internal/service"
	"xxqg-automate/internal/util"
)

func GetXxqgRedirectUrl() (ru string, err error) {
	client := util.GetClient()
	type gennerateResp struct {
		Success   bool        `json:"success"`
		ErrorCode interface{} `json:"errorCode"`
		ErrorMsg  interface{} `json:"errorMsg"`
		Result    string      `json:"result"`
		Arguments interface{} `json:"arguments"`
	}
	g := new(gennerateResp)
	_, err = client.R().SetResult(g).Get("https://login.xuexi.cn/user/qrcode/generate")
	if err != nil {
		logrus.Errorln(err.Error())
		return
	}
	logrus.Infoln(g.Result)
	codeURL := fmt.Sprintf("https://login.xuexi.cn/login/qrcommit?showmenu=false&code=%v&appId=dingoankubyrfkttorhpou", g.Result)

	code := g.Result
	ants.Submit(func() {
		type checkQrCodeResp struct {
			Code    string `json:"code"`
			Success bool   `json:"success"`
			Message string `json:"message"`
			Data    string `json:"data"`
		}

		for i := 0; i < 150; i++ {
			res := new(checkQrCodeResp)
			_, err = client.R().SetResult(res).SetFormData(map[string]string{
				"qrCode":   code,
				"goto":     "https://oa.xuexi.cn",
				"pdmToken": "",
			}).SetHeader("content-type", "application/x-www-form-urlencoded;charset=UTF-8").
				Post("https://login.xuexi.cn/login/login_with_qr")
			if err != nil {
				logrus.Error(err)
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
				logrus.Errorln(err)
				return
			}
			s2 := strings.Split(s, "=")[1]
			response, err := client.R().SetQueryParams(map[string]string{
				"code":  s2,
				"state": sign.Data.Sign + uuid.New().String(),
			}).Get("https://pc-api.xuexi.cn/login/secure_check")
			if err != nil {
				logrus.Errorln(err)
				return
			}
			user, err := GetUserInfo(response.Cookies())
			if err != nil {
				logrus.Errorln(err)
				return
			}
			user.Token = response.Cookies()[0].Value
			user.LoginTime = time.Now().Unix()
			service.UserService.UpdateByUid(context.Background(), user)
			return
		}
	})
	ru = config.GetString("xxqg.schema") + url.QueryEscape(codeURL)
	return
}
