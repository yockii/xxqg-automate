package study

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/prometheus/common/log"
	logger "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"

	"xxqg-automate/internal/constant"
	"xxqg-automate/internal/model"
	"xxqg-automate/internal/util"
)

func CheckUserCookie(cookies []*http.Cookie) (bool, error) {
	client := util.GetClient()
	response, err := client.R().SetCookies(cookies...).Get("https://pc-api.xuexi.cn/open/api/score/get")
	if err != nil {
		log.Errorln("获取用户总分错误" + err.Error())
		return true, err
	}
	if !gjson.GetBytes(response.Bytes(), "ok").Bool() {
		return false, nil
	}
	return true, nil
}

func GetToken(code, sign string) (*model.User, error) {
	resp, err := util.GetClient().R().SetQueryParams(map[string]string{
		"code":  code,
		"state": sign + uuid.New().String(),
	}).Get("https://pc-api.xuexi.cn/login/secure_check")
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	user, err := GetUserInfo(resp.Cookies())
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	var token string

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "token" {
			token = cookie.Value
			break
		}
	}
	user.Token = token
	return user, nil
}

func GetUserInfo(cookies []*http.Cookie) (*model.User, error) {
	response, err := util.GetClient().R().SetCookies(cookies...).SetHeader("Cache-Control", "no-cache").Get(constant.XxqgUrlUserInfo)
	if err != nil {
		logger.Errorln("获取用户信息失败" + err.Error())
		return nil, err
	}
	resp := response.String()
	logger.Debugln("[user] 用户信息：", resp)
	j := gjson.Parse(resp)
	uid := j.Get("data.uid").String()
	nick := j.Get("data.nick").String()

	return &model.User{
		Nick: nick,
		Uid:  uid,
	}, err
}
