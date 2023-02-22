package study

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/playwright-community/playwright-go"
	logger "github.com/sirupsen/logrus"

	"xxqg-automate/internal/constant"
	"xxqg-automate/internal/model"
)

func (c *core) startLearnVideo(user *model.User, p *playwright.Page, score *Score) (tokenFailed bool) {
	page := *p
	for i := 0; i < 20; i++ {
		links, _ := getLinks(constant.Video)
		if len(links) == 0 {
			continue
		}

		if score.Content[constant.Video] != nil && score.Content[constant.Video].CurrentScore >= score.Content[constant.Video].MaxScore && score.Content["video_time"] != nil && score.Content["video_time"].CurrentScore >= score.Content["video_time"].MaxScore {
			logger.Debugln("检测到视频学习已经完成")
			return
		} else {
			n := rand.Intn(len(links))
			_, err := page.Goto(links[n].Url, playwright.PageGotoOptions{
				Referer:   playwright.String(links[rand.Intn(len(links))].Url),
				Timeout:   playwright.Float(10000),
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			})
			if err != nil {
				logger.Errorln("页面跳转失败")
				continue
			}
			logger.Debugln("正在观看视频: ", links[n].Title)
			learnTime := 60 + rand.Intn(15) + 3
			for j := 0; j < learnTime; j++ {
				if !c.browser.IsConnected() {
					return
				}
				if rand.Float32() > 0.5 {
					ants.Submit(func() {
						_, err = page.Evaluate(fmt.Sprintf("let h = document.body.scrollHeight/120*%d;document.documentElement.scrollTop=h;", j))
						if err != nil {
							logger.Errorln("视频下滑失败")
						}
					})
				}
				time.Sleep(1 * time.Second)
			}
			score = c.Score(user)
			if score == nil || score.TotalScore == 0 {
				score, tokenFailed, _ = GetUserScore(TokenToCookies(user.Token))
				if tokenFailed {
					return
				}
			}
			if score.Content[constant.Video] != nil && score.Content[constant.Video].CurrentScore >= score.Content[constant.Video].MaxScore && score.Content["video_time"] != nil && score.Content["video_time"].CurrentScore >= score.Content["video_time"].MaxScore {
				logger.Debugln("检测到本次视频学习分数已满，退出学习")
				break
			}
		}
	}
	return
}
