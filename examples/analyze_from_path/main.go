package main

import (
	"fmt"
	"log"

	"github.com/t8y2/zhipu-video-sdk/client"
)

func main() {
	// 创建客户端，API Key 从环境变量 ZHIPU_API_KEY 读取
	vlmClient := client.NewClient("")

	// 可以调整帧率（默认为 2 fps）
	// vlmClient.SetFPS(3)

	// 视频文件路径
	videoPath := "examples/test.mp4"

	// 分析提示词
	prompt := "请分析这个视频中的内容，描述视频中发生了什么，包括场景、人物、动作等关键信息。"

	fmt.Println("正在分析视频...")

	// 使用 AnalyzeVideo 方法直接从文件路径分析
	response, err := vlmClient.AnalyzeVideo(videoPath, prompt)
	if err != nil {
		log.Fatalf("分析失败: %v", err)
	}

	fmt.Println("\n=== 分析结果 ===")
	if len(response.Choices) > 0 {
		fmt.Println(response.Choices[0].Message.Content)
	}

	fmt.Printf("\n=== Token 使用情况 ===\n")
	fmt.Printf("Prompt tokens: %d\n", response.Usage.PromptTokens)
	fmt.Printf("Completion tokens: %d\n", response.Usage.CompletionTokens)
	fmt.Printf("Total tokens: %d\n", response.Usage.TotalTokens)
}
