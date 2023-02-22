package study

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/imroc/req/v3"
	"github.com/panjf2000/ants/v2"
	"github.com/playwright-community/playwright-go"
	logger "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"

	"xxqg-automate/internal/constant"
	"xxqg-automate/internal/model"
	"xxqg-automate/internal/service"
)

const (
	ButtonDaily = `#app > div > div.layout-body > div >
div.my-points-section > div.my-points-content > div:nth-child(5) > div.my-points-card-footer > div.buttonbox > div`
)

// Score 获取积分
func (c *core) Score(user *model.User) (score *Score) {
	defer func() {
		if err := recover(); err != nil {
			logger.Errorln("获取积分异常!", err)
		}
	}()

	if !c.browser.IsConnected() {
		return
	}
	bc, err := c.browser.NewContext()
	if err != nil || bc == nil {
		logger.Errorln("创建浏览实例出错!", err)
		//TODO 退出系统重启
		os.Exit(1)
		return
	}
	// 添加一个script,防止被检测
	err = bc.AddInitScript(playwright.BrowserContextAddInitScriptOptions{
		Script: playwright.String("Object.defineProperties(navigator, {webdriver:{get:()=>undefined}});")})
	if err != nil {
		logger.Errorln("附加脚本出错!", err)
		return
	}
	defer func() {
		if err = bc.Close(); err != nil {
			logger.Errorln("关闭浏览实例出错", err)
		}
	}()
	page, err := bc.NewPage()
	if err != nil || page == nil {
		logger.Errorln("创建页面失败", err)
		//TODO 退出系统重启
		os.Exit(1)
		return
	}
	defer func() {
		_ = page.Close()
	}()
	err = bc.AddCookies(ToBrowserCookies(user.Token)...)
	if err != nil {
		logger.Errorln("添加cookies失败", err)
		return
	}
	// 跳转到积分页面
	_, err = page.Goto(constant.XxqgUrlMyPoints, playwright.PageGotoOptions{
		Referer:   playwright.String(constant.XxqgUrlMyPoints),
		Timeout:   playwright.Float(10000),
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if err != nil {
		logger.Errorln("跳转页面失败" + err.Error())
		return
	}
	// 查找总积分
	total := c.getScore(page, "#app > div > div.layout-body > div > div:nth-child(2) > div.my-points-block > span.my-points-points.my-points-red")
	today := c.getScore(page, "#app > div > div.layout-body > div > div:nth-child(2) > div.my-points-block > span:nth-child(3)")
	// article
	login := c.getScore(page, "#app > div > div.layout-body > div > div.my-points-section > div.my-points-content > div:nth-child(1) > div.my-points-card-footer > div.my-points-card-progress > div.my-points-card-text")
	article := c.getScore(page, "#app > div > div.layout-body > div > div.my-points-section > div.my-points-content > div:nth-child(2) > div.my-points-card-footer > div.my-points-card-progress > div.my-points-card-text")
	video := c.getScore(page, "#app > div > div.layout-body > div > div.my-points-section > div.my-points-content > div:nth-child(3) > div.my-points-card-footer > div.my-points-card-progress > div.my-points-card-text")
	videoTime := c.getScore(page, "#app > div > div.layout-body > div > div.my-points-section > div.my-points-content > div:nth-child(4) > div.my-points-card-footer > div.my-points-card-progress > div.my-points-card-text")
	special := c.getScore(page, "#app > div > div.layout-body > div > div.my-points-section > div.my-points-content > div:nth-child(6) > div.my-points-card-footer > div.my-points-card-progress > div.my-points-card-text")
	daily := c.getScore(page, "#app > div > div.layout-body > div > div.my-points-section > div.my-points-content > div:nth-child(5) > div.my-points-card-footer > div.my-points-card-progress > div.my-points-card-text")
	score = &Score{
		TotalScore: total,
		TodayScore: today,
		Content: map[string]*Data{
			"login": {
				CurrentScore: login,
				MaxScore:     1,
			},
			constant.Article: {
				CurrentScore: article,
				MaxScore:     12,
			},
			constant.Video: {
				CurrentScore: video,
				MaxScore:     6,
			},
			"video_time": {
				CurrentScore: videoTime,
				MaxScore:     6,
			},
			"special": {
				CurrentScore: special,
				MaxScore:     10,
			},
			"daily": {
				CurrentScore: daily,
				MaxScore:     5,
			},
		},
	}
	return
}

func (c *core) getScore(page playwright.Page, selector string) (score int) {
	div, err := page.QuerySelector(selector)
	if err != nil {
		logger.Debugln("获取积分失败", err.Error())
		return
	}
	if div == nil {
		return
	}
	str, err := div.InnerText()
	if err != nil {
		logger.Debugln("获取积分失败", err.Error())
		return
	}
	if strings.Contains(str, "分") {
		str = str[:strings.Index(str, "分")]
	}
	if str == "" {
		return
	}

	score64, _ := strconv.ParseInt(str, 10, 64)
	score = int(score64)
	return
}

// Answer 答题 1-每日 2-每周 3-专项
func (c *core) Answer(user *model.User, t int) (tokenFailed bool) {
	defer func() {
		if err := recover(); err != nil {
			logger.Errorln("答题发生异常恢复!", err)
			// 尝试重新启动
			ants.Submit(func() {
				c.Answer(user, t)
			})
		}
	}()

	score := c.Score(user)
	if score == nil || score.TotalScore == 0 {
		var err error
		score, tokenFailed, err = GetUserScore(TokenToCookies(user.Token))
		if err != nil || score == nil {
			logger.Errorln("积分获取失败，停止答题", err)
			return
		}
	}

	if !c.browser.IsConnected() {
		return
	}

	bc, err := c.browser.NewContext()
	if err != nil || bc == nil {
		logger.Errorln("创建浏览实例出错!", err)
		//TODO 退出系统重启
		os.Exit(1)
		return
	}
	// 添加一个script,防止被检测
	err = bc.AddInitScript(playwright.BrowserContextAddInitScriptOptions{
		Script: playwright.String("Object.defineProperties(navigator, {webdriver:{get:()=>undefined}});")})
	if err != nil {
		logger.Errorln("附加脚本出错!", err)
		return
	}
	defer func() {
		if err = bc.Close(); err != nil {
			logger.Errorln("关闭浏览实例出错", err)
		}
	}()

	page, err := bc.NewPage()
	if err != nil || page == nil {
		logger.Errorln("创建页面失败", err)
		//TODO 退出系统重启
		os.Exit(1)
		return
	}
	defer func() {
		_ = page.Close()
	}()
	err = bc.AddCookies(ToBrowserCookies(user.Token)...)
	if err != nil {
		logger.Errorln("添加cookies失败", err)
		return
	}
	// 跳转到积分页面
	_, err = page.Goto(constant.XxqgUrlMyPoints, playwright.PageGotoOptions{
		Referer:   playwright.String(constant.XxqgUrlMyPoints),
		Timeout:   playwright.Float(10000),
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if err != nil {
		logger.Errorln("跳转页面失败" + err.Error())
		return
	}

	switch t {
	case 1:
		// 每日答题
		{
			if score.Content["daily"].CurrentScore >= score.Content["daily"].MaxScore {
				logger.Debugln("检测到每日答题已完成，退出每日答题")
				return
			}
			err = page.Click(ButtonDaily)
			if err != nil {
				logger.Errorln("跳转每日答题出错", err)
				return
			}
		}
	case 2:
		// 每周答题
		{
			//if score.Content["weekly"].CurrentScore >= score.Content["weekly"].MaxScore {
			//	logger.Debugln("检测到每周答题已完成，退出每周答题")
			//	return
			//}
			//var id int
			//id, err = getWeekId(TokenToCookies(user.Token))
			//if err != nil {
			//	logger.Errorln("获取要答的每周答题出错", err)
			//	return
			//}
			//if id == 0 {
			//	logger.Warnln("未获取到每周答题id，退出答题")
			//	return
			//}
			//_, err = page.Goto(fmt.Sprintf(constant.XxqgUrlWeekAnswerPage, id), playwright.PageGotoOptions{
			//	Referer:   playwright.String(constant.XxqgUrlMyPoints),
			//	Timeout:   playwright.Float(10000),
			//	WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			//})
			//if err != nil {
			//	logger.Errorln("跳转每周答题出错", err)
			//	return
			//}
		}
	case 3:
		// 专项
		{
			if score.Content["special"].CurrentScore >= score.Content["special"].MaxScore {
				logger.Debugln("检测到专项答题已经完成，退出答题")
				return
			}
			var id int
			id, err = getSpecialId(TokenToCookies(user.Token))
			if err != nil {
				logger.Errorln("获取要答的专项答题出错", err)
				return
			}
			if id == 0 {
				logger.Warnln("未获取到专项答题id，退出答题")
				return
			}
			_, err = page.Goto(fmt.Sprintf(constant.XxqgUrlSpecialAnswerPage, id), playwright.PageGotoOptions{
				Referer:   playwright.String(constant.XxqgUrlMyPoints),
				Timeout:   playwright.Float(10000),
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			})
			if err != nil {
				logger.Errorln("跳转专项答题出错", err)
				return
			}
		}
	}

	// 进入答题页面
	return c.startAnswer(user, &page, score, t)
}

func (c *core) startAnswer(user *model.User, p *playwright.Page, score *Score, t int) (tokenFailed bool) {
	page := *p
	var title string
	for i := 0; i < 30; i++ {
		if !c.browser.IsConnected() {
			return
		}
		// 查看是否存在答题按钮，若按钮可用则重新提交答题
		btn, err := page.QuerySelector(`#app > div > div.layout-body > div > div.detail-body > div.action-row > button`)
		if err != nil {
			logger.Debugln("获取提交按钮失败，本次答题结束" + err.Error())
			return
		}
		if btn != nil {
			enabled, err := btn.IsEnabled()
			if err != nil {
				logger.Errorln(err.Error())
				continue
			}
			if enabled {
				err := btn.Click()
				if err != nil {
					logger.Errorln("提交答案失败")
				}
			}
		}
		// 该元素存在则说明出现了滑块
		handle, _ := page.QuerySelector("#nc_mask > div")
		if handle != nil {
			logger.Debugln(handle)
			var en bool
			en, err = handle.IsVisible()
			if err != nil {
				return
			}
			if en {
				page.Mouse().Move(496, 422)
				time.Sleep(1 * time.Second)
				page.Mouse().Down()

				page.Mouse().Move(772, 416, playwright.MouseMoveOptions{})
				page.Mouse().Up()
				time.Sleep(10 * time.Second)
				logger.Debugln("可能存在滑块")
				en, err = handle.IsVisible()
				if err != nil {
					return
				}
				if en {
					page.Evaluate("__nc.reset()")
					continue
				}
			}
		}

		switch t {
		case 1:
			{
				// 检测是否已经完成
				if score.Content["daily"] != nil && score.Content["daily"].CurrentScore >= score.Content["daily"].MaxScore {
					logger.Debugln("检测到每日答题已经完成，退出答题")
					return
				}
			}
		case 2:
			{
				// 检测是否已经完成
				if score.Content["weekly"] != nil && score.Content["weekly"].CurrentScore >= score.Content["weekly"].MaxScore {
					logger.Debugln("检测到每周答题已经完成，退出答题")
					return
				}
			}
		case 3:
			{
				// 检测是否已经完成
				if score.Content["special"] != nil && score.Content["special"].CurrentScore >= score.Content["special"].MaxScore {
					logger.Debugln("检测到专项答题已经完成，退出答题")
					return
				}
			}
		}

		// 获取题目类型
		category, err := page.QuerySelector(
			`#app > div > div.layout-body > div > div.detail-body > div.question > div.q-header`)
		if err != nil {
			logger.Errorln("没有找到题目元素", err)
			return
		}
		if category != nil {
			_ = category.WaitForElementState(`visible`)
			time.Sleep(1 * time.Second)

			// 获取题目
			var question playwright.ElementHandle
			question, err = page.QuerySelector(
				`#app > div > div.layout-body > div > div.detail-body > div.question > div.q-body > div`)
			if err != nil {
				logger.Errorln("未找到题目问题元素")
				return
			}
			// 获取题目类型
			categoryText := ""
			categoryText, err = category.TextContent()
			if err != nil {
				logger.Errorln("获取题目元素失败", err)

				return
			}
			logger.Debugln("## 题目类型：" + categoryText)

			// 获取题目的问题
			questionText := ""
			questionText, err = question.TextContent()
			if err != nil {
				logger.Errorln("获取题目问题失败", err)
				return
			}

			logger.Debugln("## 题目：" + questionText)
			if title == questionText {
				logger.Warningln("可能已经卡住，正在重试，重试次数+1")
				continue
			} else {
				i = 0
			}
			title = questionText

			// 获取答题帮助
			var openTips playwright.ElementHandle
			openTips, err = page.QuerySelector(
				`#app > div > div.layout-body > div > div.detail-body > div.question > div.q-footer > span`)
			if err != nil || openTips == nil {
				logger.Errorln("未获取到题目提示信息")
				continue
			}
			logger.Debugln("开始尝试获取打开提示信息按钮")
			// 点击提示的按钮
			err = openTips.Click()
			if err != nil {
				logger.Errorln("点击打开提示信息按钮失败", err)
				continue
			}
			logger.Debugln("已打开提示信息")
			// 获取页面内容
			content := ""
			content, err = page.Content()
			if err != nil {
				logger.Errorln("获取网页全体内容失败", err)
				continue
			}
			time.Sleep(time.Second * time.Duration(rand.Intn(3)))
			logger.Debugln("以获取网页内容")
			// 关闭提示信息
			err = openTips.Click()
			if err != nil {
				logger.Errorln("点击打开提示信息按钮失败", err)
				continue
			}
			logger.Debugln("已关闭提示信息")
			// 从整个页面内容获取提示信息
			tips := getTips(content)
			logger.Debugln("[提示信息]：", tips)

			if i > 4 {
				logger.Warningln("重试次数太多，即将退出答题")
				//options, _ := getOptions(page)
				return
			}

			// 填空题
			switch {
			case strings.Contains(categoryText, "填空题"):
				// 填充填空题
				err = FillBlank(page, tips)
				if err != nil {
					logger.Errorln("填空题答题失败", err)
					return
				}
			case strings.Contains(categoryText, "多选题"):
				logger.Debugln("读取到多选题")
				var options []string
				options, err = getOptions(page)
				if err != nil {
					logger.Errorln("获取选项失败", err)
					return
				}
				logger.Debugln("获取到选项答案：", options)
				logger.Debugln("[多选题选项]：", options)
				var answer []string

				for _, option := range options {
					for _, tip := range tips {
						if strings.Contains(strings.ReplaceAll(option, " ", ""), strings.ReplaceAll(tip, " ", "")) {
							answer = append(answer, option)
						}
					}
				}

				answer = RemoveRepByLoop(answer)

				if len(answer) < 1 {
					answer = append(answer, options...)
					logger.Debugln("无法判断答案，自动选择ABCD")
				}
				logger.Debugln("根据提示分别选择了", answer)
				// 多选题选择
				err = radioCheck(page, answer)
				if err != nil {
					return
				}
			case strings.Contains(categoryText, "单选题"):
				logger.Debugln("读取到单选题")
				var options []string
				options, err = getOptions(page)
				if err != nil {
					logger.Errorln("获取选项失败", err)
					return
				}
				logger.Debugln("获取到选项答案：", options)

				var answer []string

				if len(tips) > 1 {
					logger.Warningln("检测到单选题出现多个提示信息，即将对提示信息进行合并")
					tip := strings.Join(tips, "")
					tips = []string{tip}
				}

				for _, option := range options {
					for _, tip := range tips {
						if strings.Contains(option, tip) {
							answer = append(answer, option)
						}
					}
				}
				if len(answer) < 1 {
					answer = append(answer, options[0])
					logger.Debugln("无法判断答案，自动选择A")
				}

				logger.Debugln("根据提示分别选择了", answer)
				err = radioCheck(page, answer)
				if err != nil {
					return
				}
			}
		}

		score = c.Score(user)
		if score == nil || score.TotalScore == 0 {
			score, tokenFailed, _ = GetUserScore(TokenToCookies(user.Token))
			if tokenFailed {
				return
			}
		}
	}
	return
}

// RemoveRepByLoop 通过两重循环过滤重复元素
func RemoveRepByLoop(slc []string) []string {
	var result []string // 存放结果
	for i := range slc {
		flag := true
		for j := range result {
			if slc[i] == result[j] {
				flag = false // 存在重复元素，标识为false
				break
			}
		}
		if flag { // 标识为false，不添加进结果
			result = append(result, slc[i])
		}
	}
	return result
}

func radioCheck(page playwright.Page, answer []string) error {
	radios, err := page.QuerySelectorAll(`.q-answer.choosable`)
	if err != nil {
		logger.Errorln("获取选项失败")
		return err
	}
	logger.Debugln("获取到", len(radios), "个按钮")
	for _, radio := range radios {
		textContent := ""
		textContent, err = radio.TextContent()
		if err != nil {
			logger.Errorln("获取选项答案文本出现错误", err)
			return err
		}
		for _, s := range answer {
			if textContent == s {
				err = radio.Click()
				if err != nil {
					logger.Errorln("点击选项出现错误", err)
					return err
				}
				r := rand.Intn(2)
				time.Sleep(time.Duration(r) * time.Second)
			}
		}
	}
	r := rand.Intn(5)
	time.Sleep(time.Duration(r) * time.Second)
	checkNextBotton(page)
	return nil
}

func getOptions(page playwright.Page) ([]string, error) {
	handles, err := page.QuerySelectorAll(`.q-answer.choosable`)
	if err != nil {
		logger.Errorln("获取选项信息失败")
		return nil, err
	}
	var options []string
	for _, handle := range handles {
		content, err := handle.TextContent()
		if err != nil {
			return nil, err
		}
		options = append(options, content)
	}
	return options, err
}

func getTips(data string) []string {
	data = strings.ReplaceAll(data, " ", "")
	data = strings.ReplaceAll(data, "\n", "")
	compile := regexp.MustCompile(`<fontcolor="red">(.*?)</font>`)
	match := compile.FindAllStringSubmatch(data, -1)
	var tips []string
	for _, i := range match {
		// 新增判断提示信息为空的逻辑
		if i[1] != "" {
			tips = append(tips, i[1])
		}
	}
	return tips
}

func FillBlank(page playwright.Page, tips []string) error {
	video := false
	var answer []string
	if len(tips) < 1 {
		logger.Warningln("检测到未获取到提示信息")
		video = true
	}
	if video {
		data1, err := page.QuerySelector("#app > div > div.layout-body > div > div.detail-body > div.question > div.q-body > div > span:nth-child(1)")
		if err != nil {
			logger.Errorln("获取题目前半段失败" + err.Error())
			return err
		}
		data1Text, _ := data1.TextContent()
		logger.Debugln("题目前半段：=》" + data1Text)
		searchAnswer := service.QuestionBankService.SearchAnswer(data1Text)
		if searchAnswer != "" {
			answer = append(answer, searchAnswer)
		} else {
			answer = append(answer, "不知道")
		}
	} else {
		answer = tips
	}
	inouts, err := page.QuerySelectorAll(`div.q-body > div > input`)
	if err != nil {
		logger.Errorln("获取输入框错误" + err.Error())
		return err
	}
	logger.Debugln("获取到", len(inouts), "个填空")
	if len(inouts) == 1 && len(tips) > 1 {
		temp := ""
		for _, tip := range tips {
			temp += tip
		}
		answer = strings.Split(temp, ",")
		logger.Debugln("答案已合并处理")
	}
	var ans string
	for i := 0; i < len(inouts); i++ {
		if len(answer) < i+1 {
			ans = "不知道"
		} else {
			ans = answer[i]
		}

		err := inouts[i].Fill(ans)
		if err != nil {
			logger.Errorln("填充答案失败" + err.Error())
			continue
		}
		r := rand.Intn(4) + 1
		time.Sleep(time.Duration(r) * time.Second)
	}
	r := rand.Intn(1) + 1
	time.Sleep(time.Duration(r) * time.Second)
	checkNextBotton(page)
	return nil
}

func checkNextBotton(page playwright.Page) {
	btns, err := page.QuerySelectorAll(`#app .action-row > button`)
	if err != nil {
		logger.Errorln("未检测到按钮" + err.Error())

		return
	}
	if len(btns) <= 1 {
		err := btns[0].Click()
		if err != nil {
			logger.Errorln("点击下一题按钮失败")

			return
		}
		time.Sleep(2 * time.Second)
		_, err = btns[0].GetAttribute("disabled")
		if err != nil {
			logger.Debugln("未检测到禁言属性")

			return
		}
	} else {
		err := btns[1].Click()
		if err != nil {
			logger.Errorln("提交试卷失败")

			return
		}
		logger.Debugln("已成功提交试卷")
	}
}

type SpecialList struct {
	PageNo         int `json:"pageNo"`
	PageSize       int `json:"pageSize"`
	TotalPageCount int `json:"totalPageCount"`
	TotalCount     int `json:"totalCount"`
	List           []struct {
		TipScore    float64 `json:"tipScore"`
		EndDate     string  `json:"endDate"`
		Achievement struct {
			Score   int `json:"score"`
			Total   int `json:"total"`
			Correct int `json:"correct"`
		} `json:"achievement"`
		Year             int    `json:"year"`
		SeeSolution      bool   `json:"seeSolution"`
		Score            int    `json:"score"`
		ExamScoreId      int    `json:"examScoreId"`
		UsedTime         int    `json:"usedTime"`
		Overdue          bool   `json:"overdue"`
		Month            int    `json:"month"`
		Name             string `json:"name"`
		QuestionNum      int    `json:"questionNum"`
		AlreadyAnswerNum int    `json:"alreadyAnswerNum"`
		StartTime        string `json:"startTime"`
		Id               int    `json:"id"`
		ExamTime         int    `json:"examTime"`
		Forever          int    `json:"forever"`
		StartDate        string `json:"startDate"`
		Status           int    `json:"status"`
	} `json:"list"`
	PageNum int `json:"pageNum"`
}

func getSpecialId(cookies []*http.Cookie) (int, error) {
	c := req.C()
	c.SetCommonCookies(cookies...)
	// 获取专项答题列表
	repo, err := c.R().SetQueryParams(map[string]string{"pageSize": "1000", "pageNo": "1"}).Get(constant.XxqgUrlSpecialList)
	if err != nil {
		logger.Errorln("获取专项答题列表错误" + err.Error())
		return 0, err
	}
	dataB64, err := repo.ToString()
	if err != nil {
		logger.Errorln("获取专项答题列表获取string错误" + err.Error())
		return 0, err
	}
	// 因为返回内容使用base64编码，所以需要对内容进行转码
	data, err := base64.StdEncoding.DecodeString(gjson.Get(dataB64, "data_str").String())
	if err != nil {
		logger.Errorln("获取专项答题列表转换b64错误" + err.Error())
		return 0, err
	}
	// 创建实例对象
	list := new(SpecialList)
	// json序列号
	err = json.Unmarshal(data, list)
	if err != nil {
		logger.Errorln("获取专项答题列表转换json错误" + err.Error())
		return 0, err
	}
	logger.Debugln(fmt.Sprintf("共获取到专项答题%d个", list.TotalCount))

	// 判断是否配置选题顺序，若ReverseOrder为true则从后面选题
	//if conf.GetConfig().ReverseOrder {
	//	for i := len(list.List) - 1; i >= 0; i-- {
	//		if list.List[i].TipScore == 0 {
	//			logger.Debugln(fmt.Sprintf("获取到未答专项答题: %v，id: %v", list.List[i].Name, list.List[i].Id))
	//			return list.List[i].Id, nil
	//		}
	//	}
	//} else {
	for _, s := range list.List {
		if s.TipScore == 0 {
			logger.Debugln(fmt.Sprintf("获取到未答专项答题: %v，id: %v", s.Name, s.Id))
			return s.Id, nil
		}
	}
	//}
	logger.Warningln("你已不存在未答的专项答题了")
	return 0, errors.New("未找到专项答题")
}

type WeekList struct {
	PageNo         int `json:"pageNo"`
	PageSize       int `json:"pageSize"`
	TotalPageCount int `json:"totalPageCount"`
	TotalCount     int `json:"totalCount"`
	List           []struct {
		Month     string `json:"month"`
		Practices []struct {
			SeeSolution bool    `json:"seeSolution"`
			TipScore    float64 `json:"tipScore"`
			ExamScoreId int     `json:"examScoreId"`
			Overdue     bool    `json:"overdue"`
			Achievement struct {
				Total   int `json:"total"`
				Correct int `json:"correct"`
			} `json:"achievement"`
			Name               string `json:"name"`
			BeginYear          int    `json:"beginYear"`
			StartTime          string `json:"startTime"`
			Id                 int    `json:"id"`
			BeginMonth         int    `json:"beginMonth"`
			Status             int    `json:"status"`
			TipScoreReasonType int    `json:"tipScoreReasonType"`
		} `json:"practices"`
	} `json:"list"`
	PageNum int `json:"pageNum"`
}

func getWeekId(cookies []*http.Cookie) (int, error) {
	c := req.C()
	c.SetCommonCookies(cookies...)
	repo, err := c.R().SetQueryParams(map[string]string{"pageSize": "500", "pageNo": "1"}).Get(constant.XxqgUrlWeekList)
	if err != nil {
		logger.Errorln("获取每周答题列表错误" + err.Error())
		return 0, err
	}
	dataB64, err := repo.ToString()
	if err != nil {
		logger.Errorln("获取每周答题列表获取string错误" + err.Error())
		return 0, err
	}
	data, err := base64.StdEncoding.DecodeString(gjson.Get(dataB64, "data_str").String())
	if err != nil {
		logger.Errorln("获取每周答题列表转换b64错误" + err.Error())
		return 0, err
	}
	list := new(WeekList)
	err = json.Unmarshal(data, list)
	if err != nil {
		logger.Errorln("获取每周答题列表转换json错误" + err.Error())
		return 0, err
	}
	logger.Debugln(fmt.Sprintf("共获取到每周答题%d个", list.TotalCount))

	//if conf.GetConfig().ReverseOrder {
	//	for i := len(list.List) - 1; i >= 0; i-- {
	//		for _, practice := range list.List[i].Practices {
	//			if practice.TipScore == 0 {
	//				logger.Debugln(fmt.Sprintf("获取到未答每周答题: %v，id: %v", practice.Name, practice.Id))
	//				return practice.Id, nil
	//			}
	//		}
	//	}
	//} else {
	for _, s := range list.List {
		for _, practice := range s.Practices {
			if practice.TipScore == 0 {
				logger.Debugln(fmt.Sprintf("获取到未答每周答题: %v，id: %v", practice.Name, practice.Id))
				return practice.Id, nil
			}
		}
	}
	//}
	logger.Warningln("你已不存在未答的每周答题了")
	return 0, errors.New("未找到每周答题")
}
