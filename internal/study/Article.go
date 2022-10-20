package study

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/playwright-community/playwright-go"
	logger "github.com/sirupsen/logrus"

	"xxqg-automate/internal/constant"
	"xxqg-automate/internal/model"
)

func (c *core) Learn(user *model.User, learnModule string) {
	if !c.browser.IsConnected() {
		return
	}

	score, err := GetUserScore(TokenToCookies(user.Token))
	if err != nil {
		logger.Errorln(err)
		return
	}
	if score == nil {
		logger.Debugf("未获取到分数，结束%s学习\n", learnModule)
		return
	}
	if learnModule == constant.Article {
		if articleScore, ok := score.Content[constant.Article]; !ok || articleScore.CurrentScore >= articleScore.MaxScore {
			logger.Debugln("检测到文章学习已完成，结束学习")
			return
		}
	} else if learnModule == constant.Video {
		if videoScore, ok := score.Content[constant.Video]; !ok || (videoScore.CurrentScore >= videoScore.MaxScore && score.Content["video_time"].CurrentScore >= score.Content["video_time"].MaxScore) {
			logger.Debugln("检测到视频学习已完成，结束学习")
			return
		}
	}

	bc, err := c.browser.NewContext(playwright.BrowserNewContextOptions{
		Viewport: &playwright.BrowserNewContextOptionsViewport{
			Width:  playwright.Int(1920),
			Height: playwright.Int(1080),
		},
	})
	if err != nil {
		logger.Errorln("创建浏览实例出错! ", err)
		return
	}
	err = bc.AddInitScript(playwright.BrowserContextAddInitScriptOptions{
		Script: playwright.String("Object.defineProperties(navigator, {webdriver:{get:()=>undefined}});"),
	})
	if err != nil {
		logger.Errorln("初始化浏览实例出错! ", err)
		return
	}
	defer func() {
		if err := bc.Close(); err != nil {
			logger.Errorln("关闭浏览实例出错! ", err)
		}
	}()
	bc.AddCookies(ToBrowserCookies(user.Token)...)

	page, err := bc.NewPage()
	if err != nil {
		logger.Errorln("新建页面出错!", err)
		return
	}
	defer func() {
		if err := page.Close(); err != nil {
			logger.Errorln("关闭页面失败!", err)
		}
	}()
	switch learnModule {
	case constant.Article:
		c.startLearnArticle(user, &page, score)
	case constant.Video:
		c.startLearnVideo(user, &page, score)
	}
}

func (c *core) startLearnArticle(user *model.User, p *playwright.Page, score *Score) {
	page := *p
	for i := 0; i < 20; i++ {
		links, _ := getLinks(constant.Article)
		if len(links) == 0 {
			continue
		}
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
		logger.Debugln("正在学习文章: ", links[n].Title)
		learnTime := 60 + rand.Intn(15) + 3
		for j := 0; j < learnTime; j++ {
			if !c.browser.IsConnected() {
				return
			}
			if rand.Float32() > 0.5 {
				go func() {
					_, err = page.Evaluate(fmt.Sprintf("let h = document.body.scrollHeight/120*%d;document.documentElement.scrollTop=h;", j))
					if err != nil {
						logger.Errorln("文章下滑失败")
					}
				}()
			}
			time.Sleep(1 * time.Second)
		}
		score, _ = GetUserScore(TokenToCookies(user.Token))
		if articleScore, ok := score.Content[constant.Article]; !ok || articleScore.CurrentScore >= articleScore.MaxScore {
			logger.Debugln("检测到文章学习已完成，结束文章学习")
			return
		}
	}
}
