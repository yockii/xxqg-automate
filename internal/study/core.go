package study

import (
	"context"
	"math/rand"
	"os"
	"runtime"
	"time"

	"gitee.com/chunanyong/zorm"
	"github.com/playwright-community/playwright-go"
	"github.com/prometheus/common/log"
	logger "github.com/sirupsen/logrus"
	"github.com/yockii/qscore/pkg/config"
	"github.com/yockii/qscore/pkg/domain"

	"xxqg-automate/internal/constant"
	internalDomain "xxqg-automate/internal/domain"
	"xxqg-automate/internal/model"
	"xxqg-automate/internal/service"
	"xxqg-automate/internal/util"
)

var Core *core

type core struct {
	pw          *playwright.Playwright
	browser     playwright.Browser
	ShowBrowser bool
}

func Init() {
	Core = new(core)
	if runtime.GOOS == "windows" {
		Core.initWindows()
	} else {
		Core.initNotWindows()
	}
}

func Quit() {
	err := Core.browser.Close()
	if err != nil {
		log.Errorln("关闭浏览器失败" + err.Error())
		return
	}
	err = Core.pw.Stop()
	if err != nil {
		return
	}
}

func (c *core) initWindows() {
	path := "C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe"
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warningln("检测到edge浏览器不存在，将自动下载chrome浏览器")
			c.initNotWindows()
			return
		}
		err = nil
	}

	dir, err := os.Getwd()
	if err != nil {
		return
	}

	pwo := &playwright.RunOptions{
		DriverDirectory:     dir + "/tools/driver/",
		SkipInstallBrowsers: true,
		Browsers:            []string{"msedge"},
	}

	err = playwright.Install(pwo)
	if err != nil {
		log.Errorln("[core]", "安装playwright失败")
		log.Errorln("[core] ", err.Error())

		return
	}

	pwt, err := playwright.Run(pwo)
	if err != nil {
		log.Errorln("[core]", "初始化playwright失败")
		log.Errorln("[core] ", err.Error())

		return
	}
	c.pw = pwt
	browser, err := pwt.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Args: []string{
			"--disable-extensions",
			"--disable-gpu",
			"--start-maximized",
			"--no-sandbox",
			"--window-size=500,450",
			"--mute-audio",
			"--window-position=0,0",
			"--ignore-certificate-errors",
			"--ignore-ssl-errors",
			"--disable-features=RendererCodeIntegrity",
			"--disable-blink-features",
			"--disable-blink-features=AutomationControlled",
		},
		Channel:         nil,
		ChromiumSandbox: nil,
		Devtools:        nil,
		DownloadsPath:   playwright.String("./tools/temp/"),
		ExecutablePath:  playwright.String(path),
		HandleSIGHUP:    nil,
		HandleSIGINT:    nil,
		HandleSIGTERM:   nil,
		Headless:        playwright.Bool(!c.ShowBrowser),
		Proxy:           nil,
		SlowMo:          nil,
		Timeout:         nil,
	})
	if err != nil {
		log.Errorln("[core] ", "初始化edge失败")
		log.Errorln("[core] ", err.Error())
		return
	}

	c.browser = browser
}

func (c *core) initNotWindows() {
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	_, b := os.LookupEnv("PLAYWRIGHT_BROWSERS_PATH")
	if !b {
		err = os.Setenv("PLAYWRIGHT_BROWSERS_PATH", dir+"/tools/browser/")
		if err != nil {
			log.Errorln("设置环境变量PLAYWRIGHT_BROWSERS_PATH失败" + err.Error())
			err = nil
		}
	}

	pwo := &playwright.RunOptions{
		DriverDirectory:     dir + "/tools/driver/",
		SkipInstallBrowsers: false,
		Browsers:            []string{"chromium"},
	}

	err = playwright.Install(pwo)
	if err != nil {
		log.Errorln("[core]", "安装playwright失败")
		log.Errorln("[core] ", err.Error())

		return
	}

	pwt, err := playwright.Run(pwo)
	if err != nil {
		log.Errorln("[core]", "初始化playwright失败")
		log.Errorln("[core] ", err.Error())

		return
	}
	c.pw = pwt
	browser, err := pwt.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Args: []string{
			"--disable-extensions",
			"--disable-gpu",
			"--start-maximized",
			"--no-sandbox",
			"--window-size=500,450",
			"--mute-audio",
			"--window-position=0,0",
			"--ignore-certificate-errors",
			"--ignore-ssl-errors",
			"--disable-features=RendererCodeIntegrity",
			"--disable-blink-features",
			"--disable-blink-features=AutomationControlled",
		},
		Channel:         nil,
		ChromiumSandbox: nil,
		Devtools:        nil,
		DownloadsPath:   nil,
		ExecutablePath:  nil,
		HandleSIGHUP:    nil,
		HandleSIGINT:    nil,
		HandleSIGTERM:   nil,
		Headless:        playwright.Bool(!c.ShowBrowser),
		Proxy:           nil,
		SlowMo:          nil,
		Timeout:         nil,
	})
	if err != nil {
		log.Errorln("[core] ", "初始化chrome失败")
		log.Errorln("[core] ", err.Error())
		return
	}
	c.browser = browser
}

func isToday(d time.Time) bool {
	now := time.Now()
	return d.Year() == now.Year() && d.Month() == now.Month() && d.Day() == now.Day()
}

var studyUserMap = make(map[string]*StudyingJob)

type StudyingJob struct {
	user        *model.User
	job         *model.Job
	timer       *time.Timer
	startSignal chan bool
}

func (j *StudyingJob) startStudy(immediately ...bool) {
	studyUserMap[j.user.Id] = j
	defer func() {
		// 执行完毕将自己从map中删除
		delete(studyUserMap, j.user.Id)
	}()
	j.startSignal = make(chan bool, 1)
	if time.Time(j.user.LastStudyTime).After(time.Now()) {
		j.timer = time.NewTimer(time.Time(j.user.LastStudyTime).Sub(time.Now()))
	} else {
		var randomDuration = time.Second
		if len(immediately) == 0 || !immediately[0] {
			// 学习时间在当前时间之前
			if isToday(time.Time(j.user.LastStudyTime)) {
				// 今天的日期，随机延长120s 2分钟
				randomDuration = time.Duration(rand.Intn(120)) * time.Second
			} else {
				// 今天以前的日期，随机延长60 * 120秒 120分钟
				randomDuration = time.Duration(rand.Intn(60*120)) * time.Second

				if time.Now().Add(randomDuration).Day() != time.Now().Day() {
					randomDuration = time.Duration(rand.Intn(60)) * time.Second
				}
			}
		}

		_, err := zorm.Transaction(context.Background(), func(ctx context.Context) (interface{}, error) {
			finder := zorm.NewUpdateFinder(model.UserTableName).Append(
				"last_study_time=?, last_score=?", domain.DateTime(time.Now().Add(randomDuration)), 0).
				Append("WHERE id=?", j.user.Id)
			return zorm.UpdateFinder(ctx, finder)
		})
		if err != nil {
			logger.Errorln(err)
			return
		}

		// 随机休眠再开始学习
		j.timer = time.NewTimer(randomDuration)
	}
	select {
	case <-j.timer.C:
	case <-j.startSignal:
	}

	logger.Infoln(j.user.Nick, "开始学习")
	tokenFailed := Core.Learn(j.user, constant.Article)
	if tokenFailed {
		dealFailedToken(j.user)
		return
	}
	tokenFailed = Core.Learn(j.user, constant.Video)
	if tokenFailed {
		dealFailedToken(j.user)
		return
	}
	tokenFailed = Core.Answer(j.user, 1)
	if tokenFailed {
		dealFailedToken(j.user)
		return
	}
	tokenFailed = Core.Answer(j.user, 2)
	if tokenFailed {
		dealFailedToken(j.user)
		return
	}
	tokenFailed = Core.Answer(j.user, 3)
	if tokenFailed {
		dealFailedToken(j.user)
		return
	}

	time.Sleep(5 * time.Second)

	score, tokenFailed, err := GetUserScore(TokenToCookies(j.user.Token))
	if err != nil {
		logger.Errorln(err)
		return
	}
	if tokenFailed {
		dealFailedToken(j.user)
		return
	}
	now := domain.DateTime(time.Now())
	lastCheckTime := now

	if score == nil {
		score = &Score{}
		lastCheckTime = domain.DateTime(time.Now().Add(-time.Hour))
	}

	_, err = zorm.Transaction(context.Background(), func(ctx context.Context) (interface{}, error) {
		return zorm.UpdateNotZeroValue(ctx, &model.User{
			Id:             j.user.Id,
			LastCheckTime:  lastCheckTime,
			LastFinishTime: now,
			LastScore:      score.TodayScore,
			Score:          score.TotalScore,
		})
	})
	if err != nil {
		logger.Errorln(err)
		return
	}

	// 删除job
	service.JobService.DeleteById(context.Background(), j.job.Id)

	if config.GetString("communicate.baseUrl") != "" {
		util.GetClient().R().
			SetHeader("token", constant.CommunicateHeaderKey).
			SetBody(&internalDomain.FinishInfo{
				Nick:  j.user.Nick,
				Score: score.TodayScore,
			}).Post(config.GetString("communicate.baseUrl") + "/api/v1/finishNotify")
	}
}

func dealFailedToken(user *model.User) {
	zorm.Transaction(context.Background(), func(ctx context.Context) (interface{}, error) {
		_, err := zorm.UpdateNotZeroValue(ctx, &model.User{
			Id:            user.Id,
			LastCheckTime: domain.DateTime(time.Now()),
			Status:        -1,
		})
		if err != nil {
			return nil, err
		}
		return zorm.Delete(ctx, &model.Job{UserId: user.Id, Status: 1})
	})
	if config.GetString("communicate.baseUrl") != "" {
		if config.GetBool("xxqg.expireNotify") {
			util.GetClient().R().
				SetHeader("token", constant.CommunicateHeaderKey).
				SetBody(&internalDomain.NotifyInfo{
					Nick: user.Nick,
				}).Post(config.GetString("communicate.baseUrl") + "/api/v1/expiredNotify")
		}
	}
	return
}

func StartStudy(user *model.User, jobs ...*model.Job) {
	var job *model.Job
	if len(jobs) == 0 {
		job = &model.Job{
			UserId: user.Id,
			Status: 1,
		}
		service.JobService.DeleteByUserId(context.Background(), user.Id, 1)
		service.JobService.Save(context.Background(), job)
	} else {
		job = jobs[0]
	}
	sj := &StudyingJob{
		user: user,
		job:  job,
	}
	sj.startStudy()
}

func StartStudyRightNow(user *model.User) {
	if sj, has := studyUserMap[user.Id]; has {
		close(sj.startSignal)
		sj.timer.Stop()
	}

	job := &model.Job{
		UserId: user.Id,
		Status: 1,
	}
	service.JobService.DeleteByUserId(context.Background(), user.Id, 1)
	service.JobService.Save(context.Background(), job)
	sj := &StudyingJob{
		user: user,
		job:  job,
	}
	sj.startStudy(true)
}
