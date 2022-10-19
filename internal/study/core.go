package study

import (
	"os"
	"runtime"

	"github.com/playwright-community/playwright-go"
	"github.com/prometheus/common/log"
	logger "github.com/sirupsen/logrus"
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
