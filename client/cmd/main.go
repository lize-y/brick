package main

import (
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	cliview "github.com/lize-y/brick/client/cli-view"
	"google.golang.org/grpc"
)

func main() {
	// 连接 gRPC 服务
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// 初始化 Bubbletea 模型
	p := tea.NewProgram(cliview.InitialModel(conn))

	// 运行程序
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
