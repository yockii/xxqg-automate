package wan

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v2"
	logger "github.com/sirupsen/logrus"
	"github.com/yockii/qscore/pkg/config"
	"github.com/yockii/qscore/pkg/server"

	"xxqg-automate/internal/cache"
	"xxqg-automate/internal/constant"
	"xxqg-automate/internal/domain"
	job "xxqg-automate/internal/job/wan"
	"xxqg-automate/internal/util"
)

// 给钉钉的连接 POST /dingtalk

// 1、钉钉机器人记录谁需要连接，并在收到连接后依次通知
// 2、接收完成通知
// 3、需要统计数据
var loginReq []string
var needStatistics atomic.Bool
var bindUser = make(map[string]string) // key=钉钉id value=名字
var manualStudy []string               // 主动学习名单
var locker sync.Mutex

func InitRouter() {

	// 接收钉钉回调
	handleDingtalkCall()

	// 接收客户端定时轮询
	handleStatusAsk()

	// 接收客户端请求
	handleFinishNotify()
	handleLinkNotify()
	handleStatisticsNotify()
	handleExpiredNotify()
	handleLoginSuccessNotify()
	handleBindSuccessNotify()
	handleSendDingRequest()
}

func handleSendDingRequest() {
	server.Post("/api/v1/sendToDingtalkUser", func(ctx *fiber.Ctx) error {
		req := new(domain.SendToDingUser)
		if err := ctx.BodyParser(req); err != nil {
			logger.Errorln(err)
			return ctx.SendStatus(fiber.StatusBadRequest)
		}
		sendToDingUser(req.UserId, req.MsgKey, req.MsgParam)
		return ctx.SendStatus(fiber.StatusOK)
	})
}

func dingtalkSign(timestamp, secret string) string {
	needSign := timestamp + "\n" + secret
	signedBytes := util.HmacSha256([]byte(needSign), []byte(config.GetString("dingtalk.appSecret")))
	return base64.StdEncoding.EncodeToString(signedBytes)
}

func handleDingtalkCall() {
	type dingtalkCallBody struct {
		SenderStaffId string `json:"senderStaffId,omitempty"`
		SenderNick    string `json:"senderNick,omitempty"`
		Text          struct {
			Content string `json:"content,omitempty"`
		} `json:"text"`
	}

	server.Post("/dingtalk", func(ctx *fiber.Ctx) error {
		timestampStr := ctx.Get("timestamp")
		sign := ctx.Get("sign")
		if timestampStr == "" || sign == "" {
			return ctx.SendStatus(fiber.StatusBadRequest)
		}
		timestamp, _ := strconv.ParseInt(timestampStr, 10, 64)
		nowTimestamp := time.Now().UnixMilli()
		if nowTimestamp > timestamp+60*60*1000 || nowTimestamp < timestamp-60*60*1000 {
			return ctx.SendStatus(fiber.StatusBadRequest)
		}
		// 签名计算
		mySigned := dingtalkSign(timestampStr, config.GetString("dingtalk.appSecret"))
		if mySigned != sign {
			return ctx.SendStatus(fiber.StatusBadRequest)
		}

		// 验证确定来自钉钉的请求
		data := new(dingtalkCallBody)
		if err := ctx.BodyParser(data); err != nil {
			logger.Errorln(err)
			return ctx.SendStatus(fiber.StatusBadRequest)
		}
		if strings.Contains(data.Text.Content, "登录") {
			locker.Lock()
			defer locker.Unlock()
			loginReq = append(loginReq, data.SenderStaffId)
		} else if strings.Contains(data.Text.Content, "统计") {
			needStatistics.Store(true)
		} else if strings.Contains(data.Text.Content, "绑定") {
			nick := strings.TrimSpace(strings.ReplaceAll(data.Text.Content, "绑定 ", ""))
			if nick == "" {
				nick = data.SenderNick
			}
			bindUser[data.SenderStaffId] = nick
		} else if strings.Contains(data.Text.Content, "学习") {
			// 进行主动学习
			manualStudy = append(manualStudy, data.SenderStaffId)
		}
		return ctx.SendStatus(fiber.StatusOK)
	})
}

func handleStatusAsk() {
	server.Get("/api/v1/status", checkToken, func(ctx *fiber.Ctx) error {
		locker.Lock()
		defer locker.Unlock()
		needStudy := manualStudy
		manualStudy = []string{}
		return ctx.JSON(domain.StatusAsk{
			NeedLink:       len(loginReq) > 0,
			LinkDingId:     loginReq[0],
			NeedStatistics: needStatistics.Load(),
			BindUsers:      bindUser,
			StartStudy:     needStudy,
		})
	})
}

func checkToken(ctx *fiber.Ctx) error {
	if ctx.Get("token") != constant.CommunicateHeaderKey {
		return ctx.SendStatus(fiber.StatusForbidden)
	}
	return ctx.Next()
}

func handleStatisticsNotify() {
	server.Post("/api/v1/statisticsNotify", checkToken, func(ctx *fiber.Ctx) error {
		info := new(domain.StatisticsInfo)
		if err := ctx.BodyParser(info); err != nil {
			logger.Errorln(err)
			return ctx.SendStatus(fiber.StatusBadRequest)
		}
		needStatistics.Store(false)
		// 发送到钉钉
		sendToDingtalk("", fmt.Sprintf(
			"已完成[%d]: %s \n 学习中[%d]: %s \n 等待学习[%d]: %s \n 已失效[%d]: %s \n 未完成[%d]: %s",
			len(info.Finished),
			strings.Join(info.Finished, ","),
			len(info.Studying),
			strings.Join(info.Studying, ","),
			len(info.Waiting),
			strings.Join(info.Waiting, ","),
			len(info.Expired),
			strings.Join(info.Expired, ","),
			len(info.NotFinished),
			strings.Join(info.NotFinished, ","),
		),
			1)
		return ctx.SendStatus(fiber.StatusOK)
	})
}

func handleLinkNotify() {
	server.Post("/api/v1/newLink", checkToken, func(ctx *fiber.Ctx) error {
		l := new(domain.LinkInfo)
		if err := ctx.BodyParser(l); err != nil {
			logger.Errorln(err)
			return ctx.SendStatus(fiber.StatusBadRequest)
		}
		locker.Lock()
		defer locker.Unlock()
		atUserId := ""
		if len(loginReq) > 0 {
			atUserId = loginReq[0]
			if len(loginReq) == 1 {
				loginReq = []string{}
			} else {
				loginReq = loginReq[1:]
			}
		}
		sendToDingtalk(atUserId, l.Link, 2)
		return ctx.SendStatus(fiber.StatusOK)
	})
}

func handleFinishNotify() {
	server.Post("/api/v1/finishNotify", checkToken, func(ctx *fiber.Ctx) error {
		info := new(domain.FinishInfo)
		if err := ctx.BodyParser(info); err != nil {
			logger.Errorln(err)
			return ctx.SendStatus(fiber.StatusBadRequest)
		}
		sendToDingtalk("", fmt.Sprintf("%s已完成学习，积分：%d", info.Nick, info.Score), 1)
		return ctx.SendStatus(fiber.StatusOK)
	})
}

func handleExpiredNotify() {
	server.Post("/api/v1/expiredNotify", checkToken, func(ctx *fiber.Ctx) error {
		info := new(domain.NotifyInfo)
		if err := ctx.BodyParser(info); err != nil {
			logger.Errorln(err)
			return ctx.SendStatus(fiber.StatusBadRequest)
		}

		sendToDingtalk("", fmt.Sprintf("%s登录已失效，请重新发送登录获取登录连接", info.Nick), 1)
		return ctx.SendStatus(fiber.StatusOK)
	})
}

func handleLoginSuccessNotify() {
	server.Post("/api/v1/loginSuccessNotify", checkToken, func(ctx *fiber.Ctx) error {
		info := new(domain.NotifyInfo)
		if err := ctx.BodyParser(info); err != nil {
			logger.Errorln(err)
			return ctx.SendStatus(fiber.StatusBadRequest)
		}

		sendToDingtalk("", fmt.Sprintf("%s登录成功", info.Nick), 1)
		return ctx.SendStatus(fiber.StatusOK)
	})
}

func handleBindSuccessNotify() {
	server.Post("/api/v1/bindSuccessNotify", func(ctx *fiber.Ctx) error {
		info := new(domain.NotifyInfo)
		if err := ctx.BodyParser(info); err != nil {
			logger.Errorln(err)
			return ctx.SendStatus(fiber.StatusBadRequest)
		}
		successStr := "失败"
		if info.Success {
			successStr = "成功"
		}

		dingtalkUserId := ""

		for dingtalkId, nick := range bindUser {
			if nick == info.Nick {
				dingtalkUserId = dingtalkId
				delete(bindUser, dingtalkId)
				break
			}
		}

		sendToDingtalk(dingtalkUserId, fmt.Sprintf("%s绑定%s", info.Nick, successStr), 1)
		return ctx.SendStatus(fiber.StatusOK)
	})
}

func sendToDingtalk(atUserId string, content string, t int) {
	switch t {
	case 1: // 普通文本型
		sendCommonText(atUserId, content)
	case 2: // 链接
		sendLinkedMsg(atUserId, content)
	}
}

type DingtalkTextMessage struct {
	MsgType string `json:"msgtype,omitempty"`
	Text    struct {
		Content string `json:"content,omitempty"`
	} `json:"text"`
	At struct {
		AtMobiles []string `json:"atMobiles,omitempty"`
		AtUserIds []string `json:"atDingtalkIds,omitempty"`
		IsAtAll   bool     `json:"isAtAll,omitempty"`
	} `json:"at"`
}

func sendCommonText(atUserId string, content string) {
	apiUrl := constant.DingtalkOApiBaseUrl + "/robot/send"
	//t := fmt.Sprintf("%d", time.Now().UnixMilli())

	util.GetClient().R().
		SetHeader("Content-Type", "application/json").
		SetQueryParams(map[string]string{
			//"timestamp":    t,
			//"sign":         dingtalkSign(t, config.GetString("dingtalk.appSecret")),
			"access_token": config.GetString("dingtalk.accessToken"),
		}).
		SetBody(&DingtalkTextMessage{
			MsgType: "text",
			Text: struct {
				Content string `json:"content,omitempty"`
			}{
				Content: "[学习强国] " + content,
			},
			At: struct {
				AtMobiles []string `json:"atMobiles,omitempty"`
				AtUserIds []string `json:"atDingtalkIds,omitempty"`
				IsAtAll   bool     `json:"isAtAll,omitempty"`
			}{
				AtUserIds: []string{atUserId},
			},
		}).
		Post(apiUrl)
}

type DingtalkMarkdownMessage struct {
	MsgType  string `json:"msgtype,omitempty"`
	Markdown struct {
		Title string `json:"title"`
		Text  string `json:"text"`
	} `json:"markdown"`
	At struct {
		AtMobiles []string `json:"atMobiles,omitempty"`
		AtUserIds []string `json:"atUserIds,omitempty"`
		IsAtAll   bool     `json:"isAtAll,omitempty"`
	} `json:"at"`
}

func sendLinkedMsg(atUserId string, link string) {
	apiUrl := constant.DingtalkOApiBaseUrl + "/robot/send"
	//t := fmt.Sprintf("%d", time.Now().UnixMilli())

	_, err := util.GetClient().R().
		SetHeader("Content-Type", "application/json").
		SetQueryParams(map[string]string{
			//"timestamp":    t,
			//"sign":         dingtalkSign(t, config.GetString("dingtalk.appSecret")),
			"access_token": config.GetString("dingtalk.accessToken"),
		}).
		SetBody(&DingtalkMarkdownMessage{
			MsgType: "markdown",
			Markdown: struct {
				Title string `json:"title"`
				Text  string `json:"text"`
			}{
				Title: "学习强国登录",
				Text:  fmt.Sprintf("[学习强国] @%s [点击登录学习强国](%s)", atUserId, link),
			},
			At: struct {
				AtMobiles []string `json:"atMobiles,omitempty"`
				AtUserIds []string `json:"atUserIds,omitempty"`
				IsAtAll   bool     `json:"isAtAll,omitempty"`
			}{
				AtUserIds: []string{atUserId},
			},
		}).
		Post(apiUrl)
	if err != nil {
		logger.Errorln(err)
		return
	}
}

func sendToDingUser(userId string, msgType string, content string) {
	accessToken := ""
	found, at := cache.DefaultCache.Get(constant.DingtalkAccessToken)
	if !found {
		accessToken = job.RefreshAccessToken()
	} else {
		accessToken = at.(string)
	}
	resp, err := util.GetClient().R().SetHeader("x-acs-dingtalk-access-token", accessToken).SetBody(map[string]interface{}{
		"robotCode": config.GetString("dingtalk.appKey"),
		"userIds":   []string{userId},
		"msgKey":    msgType,
		"msgParam":  content,
	}).Post(constant.DingtalkApiBaseUrl + "/v1.0/robot/oToMessages/batchSend")
	if err != nil {
		logger.Errorln(err)
		return
	}
	logger.Debugf(resp.ToString())
}
