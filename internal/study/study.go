package study

import (
	"encoding/json"
	"errors"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/prometheus/common/log"
	logger "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"

	"xxqg-automate/internal/constant"
	"xxqg-automate/internal/util"
)

type Score struct {
	TotalScore int              `json:"total_score"`
	TodayScore int              `json:"today_score"`
	Content    map[string]*Data `json:"content"`
}
type Data struct {
	CurrentScore int `json:"current_score"`
	MaxScore     int `json:"max_score"`
}

func GetUserScore(cookies []*http.Cookie) (score *Score, tokenFailed bool, err error) {
	score = new(Score)
	var resp []byte

	header := map[string]string{
		"Cache-Control": "no-cache",
	}

	client := util.GetClient()
	response, err := client.R().SetCookies(cookies...).SetHeaders(header).Get(constant.XxqgUrlTotalScore)
	if err != nil {
		logger.Errorln("获取用户总分错误"+err.Error(), string(response.Bytes()))
		return nil, false, err
	}
	resp = response.Bytes()

	score.TotalScore = int(gjson.GetBytes(resp, "data.score").Int())

	response, err = client.R().SetCookies(cookies...).SetHeaders(header).Get(constant.XxqgUrlTodayTotalScore)
	if err != nil {
		log.Errorln("获取用户总分错误"+err.Error(), string(response.Bytes()))
		return nil, false, err
	}
	resp = response.Bytes()
	score.TodayScore = int(gjson.GetBytes(resp, "data.score").Int())

	response, err = client.R().SetCookies(cookies...).SetHeaders(header).Get(constant.XxqgUrlRateScore)
	if err != nil {
		log.Errorln("获取用户总分错误"+err.Error(), string(response.Bytes()))
		return nil, false, err
	}
	resp = response.Bytes()
	j := gjson.ParseBytes(resp)
	taskProgress := j.Get("data.taskProgress").Array()
	if len(taskProgress) == 0 {
		if j.Get("code").Int() == 401 {
			// 校验失败
			return nil, true, nil
		}

		logger.Warnln("未获取到data.taskProgress: ", string(resp))
		return nil, false, errors.New("未成功获取用户积分信息")
	}
	m := make(map[string]*Data, 7)
	m[constant.Article] = &Data{
		CurrentScore: int(taskProgress[0].Get("currentScore").Int()),
		MaxScore:     int(taskProgress[0].Get("dayMaxScore").Int()),
	}
	m[constant.Video] = &Data{
		CurrentScore: int(taskProgress[1].Get("currentScore").Int()),
		MaxScore:     int(taskProgress[1].Get("dayMaxScore").Int()),
	}
	m["video_time"] = &Data{
		CurrentScore: int(taskProgress[2].Get("currentScore").Int()),
		MaxScore:     int(taskProgress[2].Get("dayMaxScore").Int()),
	}
	m["login"] = &Data{
		CurrentScore: int(taskProgress[3].Get("currentScore").Int()),
		MaxScore:     int(taskProgress[3].Get("dayMaxScore").Int()),
	}
	m["special"] = &Data{
		CurrentScore: int(taskProgress[4].Get("currentScore").Int()),
		MaxScore:     int(taskProgress[4].Get("dayMaxScore").Int()),
	}
	m["daily"] = &Data{
		CurrentScore: int(taskProgress[5].Get("currentScore").Int()),
		MaxScore:     int(taskProgress[5].Get("dayMaxScore").Int()),
	}

	score.Content = m
	return
}

type Link struct {
	Editor       string   `json:"editor"`
	PublishTime  string   `json:"publishTime"`
	ItemType     string   `json:"itemType"`
	Author       string   `json:"author"`
	CrossTime    int      `json:"crossTime"`
	Source       string   `json:"source"`
	NameB        string   `json:"nameB"`
	Title        string   `json:"title"`
	Type         string   `json:"type"`
	Url          string   `json:"url"`
	ShowSource   string   `json:"showSource"`
	ItemId       string   `json:"itemId"`
	ThumbImage   string   `json:"thumbImage"`
	AuditTime    string   `json:"auditTime"`
	ChannelNames []string `json:"channelNames"`
	Producer     string   `json:"producer"`
	ChannelIds   []string `json:"channelIds"`
	DataValid    bool     `json:"dataValid"`
}

func getLinks(model string) ([]Link, error) {
	UID := rand.Intn(20000000) + 10000000
	learnUrl := ""
	if model == constant.Article {
		learnUrl = constant.ArticleUrlList[rand.Intn(len(constant.ArticleUrlList))]
	} else if model == constant.Video {
		learnUrl = constant.VideoUrlList[rand.Intn(len(constant.VideoUrlList))]
	} else {
		return nil, errors.New("获取连接模块不支持")
	}
	var (
		resp []byte
	)

	response, err := util.GetClient().R().SetQueryParam("_st", strconv.Itoa(UID)).Get(learnUrl)
	if err != nil {
		logger.Errorln("请求链接列表出现错误！" + err.Error())
		return nil, err
	}
	resp = response.Bytes()

	var links []Link
	err = json.Unmarshal(resp, &links)
	if err != nil {
		logger.Errorln("解析列表出现错误" + err.Error())
		return nil, err
	}
	return links, err
}

func TokenToCookies(token string) []*http.Cookie {
	cookie := &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		Domain:   "xuexi.cn",
		Expires:  time.Now().Add(time.Hour * 12),
		Secure:   false,
		HttpOnly: false,
		SameSite: http.SameSiteStrictMode,
	}
	return []*http.Cookie{cookie}
}

func ToBrowserCookies(token string) []playwright.BrowserContextAddCookiesOptionsCookies {
	cookie := playwright.BrowserContextAddCookiesOptionsCookies{
		Name:     playwright.String("token"),
		Value:    playwright.String(token),
		Path:     playwright.String("/"),
		Domain:   playwright.String(".xuexi.cn"),
		Expires:  playwright.Float(float64(time.Now().Add(time.Hour * 12).Unix())),
		Secure:   playwright.Bool(false),
		HttpOnly: playwright.Bool(false),
		SameSite: playwright.SameSiteAttributeStrict,
	}
	return []playwright.BrowserContextAddCookiesOptionsCookies{cookie}
}
