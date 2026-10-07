package main

import (
	"fmt"
	"os"
	"runtime"

	"Four-PlayerArmyChess/gui"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	// 创建日志文件
	logFile, err := os.Create("game.log")
	if err != nil {
		fmt.Println("无法创建日志文件:", err)
		os.Exit(1)
	}
	defer logFile.Close()

	// 重定向标准输出和标准错误到日志文件
	os.Stdout = logFile
	os.Stderr = logFile

	fmt.Println("程序启动...")

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("程序发生panic:", r)
			os.Exit(1)
		}
	}()

	gui.Run()

	fmt.Println("程序正常结束")
}
