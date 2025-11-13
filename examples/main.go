package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/t8y2/zhipu-video-sdk/client"
	"github.com/t8y2/zhipu-video-sdk/processor"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <h264_file_path> [prompt]")
		fmt.Println("Example: go run main.go video.h264 \"请描述这个视频中发生了什么\"")
		os.Exit(1)
	}

	h264Path := os.Args[1]
	prompt := "请详细描述这个视频中的内容、场景和主要活动。"
	if len(os.Args) > 2 {
		prompt = os.Args[2]
	}

	// 创建客户端
	c := client.NewClient("")
	if c.APIKey == "" {
		log.Fatal("请设置 ZHIPU_API_KEY 环境变量")
	}

	// 配置流处理器
	// FPS: 2, 分辨率: 1120x1120, 质量: 90
	c.ConfigureStreamProcessor(2, 1120, 1120, 90)

	// 可选：如果知道视频的 SPS/PPS 参数，可以设置
	// c.SetStreamSPSPPS("your_sps_base64", "your_pps_base64")

	fmt.Printf("正在读取 H.264 文件: %s\n", h264Path)

	// 读取 H.264 文件
	h264Data, err := os.ReadFile(h264Path)
	if err != nil {
		log.Fatalf("读取 H.264 文件失败: %v", err)
	}

	fmt.Printf("H.264 数据大小: %d 字节\n", len(h264Data))
	fmt.Println("正在处理 H.264 视频流并提取帧...")

	start := time.Now()

	// 分析 H.264 流
	resp, err := c.AnalyzeH264Stream(h264Data, prompt)
	if err != nil {
		log.Fatalf("分析失败: %v", err)
	}

	elapsed := time.Since(start)

	// 清理临时文件
	if err := c.CleanupStreamProcessor(); err != nil {
		log.Printf("清理临时文件失败: %v", err)
	}

	// 输出结果
	fmt.Printf("\n处理耗时: %v\n", elapsed)
	fmt.Printf("模型: %s\n", resp.Model)
	fmt.Printf("使用 Token: %d (输入: %d, 输出: %d)\n",
		resp.Usage.TotalTokens,
		resp.Usage.PromptTokens,
		resp.Usage.CompletionTokens)
	fmt.Println("\n分析结果:")
	fmt.Println("----------------------------------------")

	if len(resp.Choices) > 0 {
		fmt.Println(resp.Choices[0].Message.Content)
	} else {
		fmt.Println("未获取到分析结果")
	}
}

// 示例 2: 实时流处理
func exampleRealtimeStreamProcessing() {
	// 创建流处理器
	streamProcessor := processor.NewStreamProcessor()
	streamProcessor.WithFPS(2).WithResolution(1120, 1120).WithQuality(90)

	// 创建流帧提取器
	extractor := processor.NewStreamFrameExtractor(streamProcessor)

	// 假设这是一个网络流或管道
	var streamReader io.Reader // 从网络、文件或其他来源读取

	// 开始处理流
	extractor.Start(streamReader)

	// 创建客户端
	c := client.NewClient("")

	// 在后台处理提取的帧
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	go func() {
		frameCount := 0
		for {
			select {
			case frame, ok := <-extractor.GetFrameChannel():
				if !ok {
					fmt.Println("帧通道已关闭")
					return
				}
				frameCount++
				fmt.Printf("接收到第 %d 帧，大小: %d 字节\n", frameCount, len(frame))

				// 每 10 帧分析一次
				if frameCount%10 == 0 {
					frames := [][]byte{frame}
					resp, err := c.AnalyzeFrames("描述这一帧的内容", frames)
					if err != nil {
						log.Printf("分析失败: %v", err)
					} else if len(resp.Choices) > 0 {
						fmt.Printf("分析结果: %s\n", resp.Choices[0].Message.Content)
					}
				}

			case err, ok := <-extractor.GetErrorChannel():
				if !ok {
					return
				}
				log.Printf("处理错误: %v", err)

			case <-ctx.Done():
				fmt.Println("处理超时")
				extractor.Stop()
				return
			}
		}
	}()

	// 等待处理完成
	<-ctx.Done()
	extractor.Stop()
}
