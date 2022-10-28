package wan

import (
	logger "github.com/sirupsen/logrus"
	"github.com/yockii/qscore/pkg/config"
	"github.com/yockii/qscore/pkg/task"

	"xxqg-automate/internal/cache"
	"xxqg-automate/internal/constant"
	"xxqg-automate/internal/util"
)

type atReq struct {
	AppKey    string `json:"appKey,omitempty"`
	AppSecret string `json:"appSecret,omitempty"`
}

type atResp struct {
	AccessToken string `json:"accessToken,omitempty"`
	ExpireIn    int64  `json:"expireIn,omitempty"`
}

func init() {
	task.AddFunc("@every 1h", func() { RefreshAccessToken() })
	RefreshAccessToken()
}

func RefreshAccessToken() string {
	resp := new(atResp)
	response, err := util.GetClient().R().SetBody(&atReq{
		AppKey:    config.GetString("dingtalk.appKey"),
		AppSecret: config.GetString("dingtalk.appSecret"),
	}).SetResult(resp).Post(constant.DingtalkApiBaseUrl + "/v1.0/oauth2/accessToken")
	if err != nil {
		logger.Errorln(err)
		return ""
	}
	logger.Debugln(response.String())
	cache.DefaultCache.Set(constant.DingtalkAccessToken, resp.AccessToken, resp.ExpireIn)
	return resp.AccessToken
}
