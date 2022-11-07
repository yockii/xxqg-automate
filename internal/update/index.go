package update

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/kardianos/osext"
	logger "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

func SelfUpdate(github string, version string) {
	if github == "" {
		github = "https://github.com"
	}

	if version == "unknown" {
		logger.Warningln("测试版本，不更新！")
		return
	}

	logger.Infoln("正在检查更新.")
	latest, err := lastVersion()
	if err != nil {
		logger.Warnf("获取最新版本失败: %v \n", err)
		wait()
	}
	url := fmt.Sprintf("%v/yockii/xxqg-automate/releases/download/%v/%v", github, latest, binaryName())
	if version == latest {
		logger.Infoln("当前版本已经是最新版本!")
		wait()
	}
	logger.Infoln("当前最新版本为 ", latest)
	logger.Infoln("正在更新,请稍等...")
	sum := checksum(github, latest)
	if sum != nil {
		err = update(url, sum)
		if err != nil {
			logger.Errorln("更新失败: ", err)
		} else {
			logger.Infoln("更新成功!")
		}
	} else {
		logger.Errorln("checksum 失败!")
	}
}
func checksum(github, version string) []byte {
	sumURL := fmt.Sprintf("%v/yockii/xxqg-automate/releases/download/%v/xxqg-automate_checksums.txt", github, version)
	response, err := http.Get(sumURL)
	if err != nil {
		return nil
	}
	rd := bufio.NewReader(response.Body)
	for {
		str, err := rd.ReadString('\n')
		if err != nil {
			break
		}
		str = strings.TrimSpace(str)
		if strings.HasSuffix(str, binaryName()) {
			sum, _ := hex.DecodeString(strings.TrimSuffix(str, "  "+binaryName()))
			return sum
		}
	}
	return nil
}

func binaryName() string {
	goarch := runtime.GOARCH
	if goarch == "arm" {
		goarch += "v7"
	}
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("xxqg-automate_%v_%v.%v", runtime.GOOS, goarch, ext)
}

func wait() {
	logger.Info("按 Enter 继续....")
	readLine()
	os.Exit(0)
}

func readLine() (str string) {
	console := bufio.NewReader(os.Stdin)
	str, _ = console.ReadString('\n')
	str = strings.TrimSpace(str)
	return
}
func CheckUpdate(version string) string {
	logger.Infoln("正在检查更新.")
	if version == "(devel)" {
		logger.Warnln("检查更新失败: 使用的 Actions 测试版或自编译版本.")
		return ""
	}
	if version == "unknown" {
		logger.Warnln("检查更新失败: 使用的未知版本.")
		return ""
	}

	if !strings.HasPrefix(version, "v") {
		logger.Warnln("版本格式错误")
		return ""
	}
	latest, err := lastVersion()
	if err != nil {
		logger.Warnf("检查更新失败: %v \n", err)
		return ""
	}
	if versionCompare(version, latest) {
		logger.Infoln("当前有更新的 xxqg-automate 可供更新, 请前往 https://github.com/yockii/xxqg-automate/releases 下载.")
		logger.Infof("当前版本: %v 最新版本: %v \n", version, latest)
		return "检测到可用更新，版本号：" + latest
	}
	logger.Infoln("检查更新完成. 当前已运行最新版本.")
	return ""
}

func lastVersion() (string, error) {
	response, err := http.Get("https://api.github.com/repos/yockii/xxqg-automate/releases/latest")
	if err != nil {
		return "", err
	}
	data, _ := io.ReadAll(response.Body)
	defer response.Body.Close()
	return gjson.GetBytes(data, "tag_name").Str, nil
}
func versionCompare(nowVersion, lastVersion string) bool {
	NowBeta := strings.Contains(nowVersion, "beta")
	LastBeta := strings.Contains(lastVersion, "beta")

	// 获取主要版本号
	nowMainVersion := strings.Split(nowVersion, "-")
	lastMainVersion := strings.Split(lastVersion, "-")

	nowMainIntVersion, _ := strconv.Atoi(strings.ReplaceAll(strings.TrimLeft(nowMainVersion[0], "v"), ".", ""))
	lastMainIntVersion, _ := strconv.Atoi(strings.ReplaceAll(strings.TrimLeft(lastMainVersion[0], "v"), ".", ""))

	if nowMainIntVersion < lastMainIntVersion {
		return true
	}
	if strings.Contains(nowVersion, "SNAPSHOT") {
		if nowMainIntVersion == lastMainIntVersion {
			return false
		} else {
			return true
		}
	}
	// 如果最新版本是beta
	if LastBeta {
		// 如果当前版本也是beta
		if NowBeta {
			// 对beta后面的数字进行比较
			nowBetaVersion, _ := strconv.Atoi(strings.TrimLeft(nowMainVersion[1], "beta"))
			lastBetaVersion, _ := strconv.Atoi(strings.TrimLeft(lastMainVersion[1], "beta"))
			if nowBetaVersion < lastBetaVersion {
				return true
			}
			return false
			// 如果当前版本部署beta,需要更新
		} else {
			return true
		}
		// 最新版本不是beta,需要更新
	} else {
		return false
	}
}

func fromStream(updateWith io.Reader) (err error, errRecover error) {
	updatePath, err := osext.Executable()
	if err != nil {
		return
	}

	// get the directory the executable exists in
	updateDir := filepath.Dir(updatePath)
	filename := filepath.Base(updatePath)
	// Copy the contents of of newbinary to a the new executable file
	newPath := filepath.Join(updateDir, fmt.Sprintf(".%s.new", filename))
	fp, err := os.OpenFile(newPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return
	}
	// We won't log this error, because it's always going to happen.
	defer func() { _ = fp.Close() }()
	if _, err = io.Copy(fp, bufio.NewReader(updateWith)); err != nil {
		logger.Errorf("Unable to copy data: %v\n", err)
	}

	// if we don't call fp.Close(), windows won't let us move the new executable
	// because the file will still be "in use"
	if err = fp.Close(); err != nil {
		logger.Errorf("Unable to close file: %v\n", err)
	}
	// this is where we'll move the executable to so that we can swap in the updated replacement
	oldPath := filepath.Join(updateDir, fmt.Sprintf(".%s.old", filename))

	// delete any existing old exec file - this is necessary on Windows for two reasons:
	// 1. after a successful update, Windows can't remove the .old file because the process is still running
	// 2. windows rename operations fail if the destination file already exists
	_ = os.Remove(oldPath)

	// move the existing executable to a new file in the same directory
	err = os.Rename(updatePath, oldPath)
	if err != nil {
		return
	}

	// move the new executable in to become the new program
	err = os.Rename(newPath, updatePath)

	if err != nil {
		// copy unsuccessful
		errRecover = os.Rename(oldPath, updatePath)
	} else {
		// copy successful, remove the old binary
		_ = os.Remove(oldPath)
	}
	return
}

type writeSumCounter struct {
	total uint64
	hash  hash.Hash
}

func (wc *writeSumCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.total += uint64(n)
	wc.hash.Write(p)
	fmt.Printf("\r                                    ")
	fmt.Printf("\rDownloading... %s complete", humanize.Bytes(wc.total))
	return n, nil
}
