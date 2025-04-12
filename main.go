package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"process-guardian/guardian"
	"runtime"
	"syscall"

	"github.com/getlantern/systray"
)

var (
	isPaused bool = false
	daemon   *guardian.Daemon
)

func init() {
	// 确保在 Windows 下运行时使用正确的线程模型
	runtime.LockOSThread()
}
 

func main() {  
	// 启动系统托盘
	systray.Run(onReady, onExit)
}

func onReady() {
	// 加载并设置图标
	icon, iconErr := ioutil.ReadFile("icon.ico")
	if iconErr != nil {
		log.Printf("Failed to load icon: %v", iconErr)
	} else {
		systray.SetIcon(icon)
	}
	systray.SetTooltip("Process Guardian - 进程守护")

	// 添加菜单项
	mPause := systray.AddMenuItem("暂停", "暂停守护进程")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("退出", "退出程序")

	// 初始化守护进程
	var daemonErr error
	daemon, daemonErr = guardian.NewDaemon("config.json")
	if daemonErr != nil {
		log.Printf("Failed to initialize daemon: %v", daemonErr)
		os.Exit(1)
	}

	// 启动守护进程
	if err := daemon.Start(); err != nil {
		log.Printf("Failed to start daemon: %v", err)
		os.Exit(1)
	}

	// 处理暂停/开始按钮点击
	go func() {
		for {
			<-mPause.ClickedCh
			if isPaused {
				// 恢复运行
				if err := daemon.Resume(); err != nil {
					log.Printf("Failed to resume daemon: %v", err)
					continue
				}
				mPause.SetTitle("暂停")
				isPaused = false
			} else {
				// 暂停运行
				daemon.Pause()
				mPause.SetTitle("开始")
				isPaused = true
			}
		}
	}()

	// 处理退出按钮点击
	go func() {
		<-mQuit.ClickedCh
		daemon.Stop()
		systray.Quit()
	}()

	// 同时也处理系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		daemon.Stop()
		systray.Quit()
	}()
}

func onExit() {
	if daemon != nil {
		daemon.Stop()
	}
}