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

	"xxqg-automate/internal/constant"
	"xxqg-automate/internal/domain/wan"
	"xxqg-automate/internal/util"
)

// 给钉钉的连接 POST /dingtalk

// 1、钉钉机器人记录谁需要连接，并在收到连接后依次通知
// 2、接收完成通知
// 3、需要统计数据
var loginReq []string
var needStatistics atomic.Bool
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
}

func dingtalkSign(timestamp, secret string) string {
	needSign := timestamp + "\n" + secret
	signedBytes := util.HmacSha256([]byte(needSign), []byte(config.GetString("dingtalk.appSecret")))
	return base64.StdEncoding.EncodeToString(signedBytes)
}

func handleDingtalkCall() {
	type dingtalkCallBody struct {
		SenderStaffId string `json:"senderStaffId,omitempty"`
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
		} else if strings.Contains(data.Text.Content, "测试") {
			sendCommonText(data.SenderStaffId, "测试")
		}
		return ctx.SendStatus(fiber.StatusOK)
	})
}

func handleStatusAsk() {
	server.Get("/api/v1/status", checkToken, func(ctx *fiber.Ctx) error {
		return ctx.JSON(wan.StatusAsk{
			NeedLink:       len(loginReq) > 0,
			NeedStatistics: needStatistics.Load(),
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
		info := new(wan.StatisticsInfo)
		if err := ctx.BodyParser(info); err != nil {
			logger.Errorln(err)
			return ctx.SendStatus(fiber.StatusBadRequest)
		}
		needStatistics.Store(false)
		// 发送到钉钉
		sendToDingtalk("", fmt.Sprintf(
			"已完成: %s \n 学习中: %s \n 等待学习: %s \n 已失效: %s",
			strings.Join(info.Finished, ","),
			strings.Join(info.Studying, ","),
			strings.Join(info.Waiting, ","),
			strings.Join(info.Expired, ","),
		),
			1)
		return ctx.SendStatus(fiber.StatusOK)
	})
}

func handleLinkNotify() {
	server.Post("/api/v1/newLink", checkToken, func(ctx *fiber.Ctx) error {
		l := new(wan.LinkInfo)
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
		info := new(wan.FinishInfo)
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
		info := new(wan.ExpiredInfo)
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
		info := new(wan.ExpiredInfo)
		if err := ctx.BodyParser(info); err != nil {
			logger.Errorln(err)
			return ctx.SendStatus(fiber.StatusBadRequest)
		}

		sendToDingtalk("", fmt.Sprintf("%s登录成功", info.Nick), 1)
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
