package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/panjf2000/ants/v2"
	logger "github.com/sirupsen/logrus"
	"github.com/yockii/qscore/pkg/config"
	"github.com/yockii/qscore/pkg/server"
	"github.com/yockii/qscore/pkg/task"

	"xxqg-automate/internal/controller/wan"
	_ "xxqg-automate/internal/job/wan"
)

var daemon bool

func init() {
	config.DefaultInstance.SetDefault("server.port", 8080)

	flag.BoolVar(&daemon, "daemon", false, "以守护进程方式启动")
	flag.Parse()
}

func main() {

	if daemon {
		runDaemon(os.Args)
		return
	}

	defer ants.Release()

	task.Start()
	defer task.Stop()

	wan.InitRouter()
	logger.Error(server.Start(":" + config.GetString("server.port")))
}

// 以守护进程方式启动
func runDaemon(args []string) {
	fmt.Printf("pid:%d ppid: %d, arg: %s \n", os.Getpid(), os.Getppid(), os.Args)
	// 去除--daemon参数，启动主程序
	for i := 0; i < len(args); {
		if args[i] == "--daemon" && i != len(args)-1 {
			args = append(args[:i], args[i+1:]...)
		} else if args[i] == "--daemon" && i == len(args)-1 {
			args = args[:i]
		} else {
			i++
		}
	}
	// 启动子进程
	for {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Start()
		if err != nil {
			fmt.Fprintf(os.Stderr, "启动失败, Error: %s \n", err)
			return
		}
		fmt.Printf("守护进程模式启动, pid:%d ppid: %d, arg: %s \n", cmd.Process.Pid, os.Getpid(), args)
		cmd.Wait()
	}
}
