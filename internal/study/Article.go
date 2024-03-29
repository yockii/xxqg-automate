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

func (c *core) Learn(user *model.User, learnModule string) (tokenFailed bool) {
	if !c.browser.IsConnected() {
		return
	}
	score := c.Score(user)
	if score == nil || score.TotalScore == 0 {
		logger.Warnf("未能成功获取到用户%s的积分，停止学习", user.Nick)
		return
	}
	if score == nil {
		logger.Debugf("未获取到分数，结束%s学习\n", learnModule)
		return
	}
	if learnModule == constant.Article {
		if articleScore, ok := score.Content[constant.Article]; !ok || articleScore.CurrentScore >= articleScore.MaxScore {
			logger.Debugf("%s 检测到文章学习已完成，结束学习", user.Nick)
			return
		}
	} else if learnModule == constant.Video {
		if videoScore, ok := score.Content[constant.Video]; !ok || (videoScore.CurrentScore >= videoScore.MaxScore && score.Content["video_time"].CurrentScore >= score.Content["video_time"].MaxScore) {
			logger.Debugf("%s 检测到视频学习已完成，结束学习", user.Nick)
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
		logger.Errorf("%s创建浏览实例出错! %s", user.Nick, err)
		return
	}
	err = bc.AddInitScript(playwright.BrowserContextAddInitScriptOptions{
		Script: playwright.String("Object.defineProperties(navigator, {webdriver:{get:()=>undefined}});"),
	})
	if err != nil {
		logger.Errorf("%s 初始化浏览实例出错! %s", user.Nick, err)
		return
	}
	defer func() {
		if err := bc.Close(); err != nil {
			logger.Errorf("%s关闭浏览实例出错! %s", user.Nick, err)
		}
	}()
	bc.AddCookies(ToBrowserCookies(user.Token)...)

	page, err := bc.NewPage()
	if err != nil {
		logger.Errorf("%s新建页面出错! %s", user.Nick, err)
		return
	}
	defer func() {
		if err := page.Close(); err != nil {
			logger.Errorf("%s关闭页面失败! %s", user.Nick, err)
		}
	}()
	switch learnModule {
	case constant.Article:
		c.startLearnArticle(user, &page, score)
	case constant.Video:
		c.startLearnVideo(user, &page, score)
	}
	return
}

func (c *core) startLearnArticle(user *model.User, p *playwright.Page, score *Score) (tokenFailed bool) {
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
			logger.Errorf("%s页面跳转失败", user.Nick)
			continue
		}
		logger.Debugf("%s 正在学习文章: %s", user.Nick, links[n].Title)
		learnTime := 60 + rand.Intn(15) + 3
		for j := 0; j < learnTime; j++ {
			if !c.browser.IsConnected() {
				return
			}
			if rand.Float32() > 0.5 {
				go func() {
					_, err = page.Evaluate(fmt.Sprintf("let h = document.body.scrollHeight/120*%d;document.documentElement.scrollTop=h;", j))
					if err != nil {
						logger.Errorf("%s 文章下滑失败", user.Nick)
					}
				}()
			}

			time.Sleep(((time.Duration)(200 + rand.Intn(800))) * time.Millisecond)
		}
		score = c.Score(user)
		if score == nil || score.TotalScore == 0 {
			logger.Warnf("未能成功获取到用户%s的积分，停止学习", user.Nick)
			return
		}
		if articleScore, ok := score.Content[constant.Article]; !ok || articleScore.CurrentScore >= articleScore.MaxScore {
			logger.Debugf("%s 检测到文章学习已完成，结束文章学习", user.Nick)
			return
		}
	}
	return
}
