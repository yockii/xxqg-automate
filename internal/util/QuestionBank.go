package util

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"

	logger "github.com/sirupsen/logrus"
)

var (
	dbSum = "d6e455f03b419af108cced07ea1d17f8268400ad1b6d80cb75d58e952a5609bf"
)

func CheckQuestionDB() bool {

	if !FileIsExist("./conf/QuestionBank.db") {
		return false
	}
	f, err := os.Open("./conf/QuestionBank.db")
	if err != nil {
		logger.Errorln(err.Error())
		return false
	}

	defer f.Close()
	h := sha256.New()
	//h := sha1.New()
	//h := sha512.New()

	if _, err = io.Copy(h, f); err != nil {
		logger.Errorln(err.Error())
		return false
	}

	// 格式化为16进制字符串
	sha := fmt.Sprintf("%x", h.Sum(nil))
	logger.Infoln("db_sha: " + sha)
	if sha != dbSum {
		return false
	}
	return true
}

func DownloadDbFile() {
	defer func() {
		err := recover()
		if err != nil {
			logger.Errorln("下载题库文件意外错误")
			logger.Errorln(err)
		}
	}()
	logger.Infoln("正在从github下载题库文件！")
	response, err := http.Get("https://github.com/johlanse/study_xxqg/releases/download/v1.0.36/QuestionBank.db")
	if err != nil {
		logger.Errorln("下载db文件错误" + err.Error())
		return
	}
	data, _ := io.ReadAll(response.Body)
	err = os.WriteFile("./QuestionBank.db", data, 0666)
	if err != nil {
		logger.Errorln(err.Error())
		return
	}
}
