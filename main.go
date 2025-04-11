package main

import (
	"os"
	"os/signal"
	"syscall"
	"io/ioutil"
	"github.com/getlantern/systray"
	"process-guardian/guardian"
	"runtime"
)

var (
	isPaused bool = false
	daemon   *guardian.Daemon
)

func init() {
	// 确保在 Windows 下运行时使用正确的线程模型
	runtime.LockOSThread()
}

func hideConsole() {
	if runtime.GOOS == "windows" {
		// 获取当前进程的控制台窗口
		kernel32 := syscall.NewLazyDLL("kernel32.dll")
		user32 := syscall.NewLazyDLL("user32.dll")
		getConsoleWindow := kernel32.NewProc("GetConsoleWindow")
		showWindow := user32.NewProc("ShowWindow")
		
		hwnd, _, _ := getConsoleWindow.Call()
		if hwnd != 0 {
			showWindow.Call(hwnd, 0) // SW_HIDE = 0
		}
	}
}

func main() {
	// 隐藏控制台窗口
	hideConsole()
	
	// 启动系统托盘
	systray.Run(onReady, onExit)
}

func onReady() {
	// 加载并设置图标
	if icon, err := ioutil.ReadFile("icon.ico"); err == nil {
		systray.SetIcon(icon)
	}
	systray.SetTooltip("Process Guardian - 进程守护")

	// 添加菜单项
	mPause := systray.AddMenuItem("暂停", "暂停守护进程")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("退出", "退出程序")

	// 初始化守护进程
	var err error
	daemon, err = guardian.NewDaemon("config.json")
	if err != nil {
		os.Exit(1)
	}

	// 启动守护进程
	if err := daemon.Start(); err != nil {
		os.Exit(1)
	}

	// 处理暂停/开始按钮点击
	go func() {
		for {
			<-mPause.ClickedCh
			if isPaused {
				// 恢复运行
				daemon.Start()
				mPause.SetTitle("暂停")
				isPaused = false
			} else {
				// 暂停运行
				daemon.Stop()
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