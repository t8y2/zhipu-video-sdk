package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// GenerateH264FromVideo 从普通视频文件生成 H.264 原始流文件
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run generate_h264.go <input_video> [output.h264]")
		fmt.Println("Example: go run generate_h264.go input.mp4 output.h264")
		fmt.Println("\n这个工具将普通视频文件转换为 H.264 原始流格式")
		os.Exit(1)
	}

	inputVideo := os.Args[1]
	outputH264 := "output.h264"
	if len(os.Args) > 2 {
		outputH264 = os.Args[2]
	}

	// 检查输入文件是否存在
	if _, err := os.Stat(inputVideo); os.IsNotExist(err) {
		log.Fatalf("输入文件不存在: %s", inputVideo)
	}

	fmt.Printf("输入视频: %s\n", inputVideo)
	fmt.Printf("输出 H.264: %s\n", outputH264)

	// 使用 ffmpeg 将视频转换为 H.264 原始流
	// -vcodec copy: 如果输入已经是 H.264，直接复制
	// -an: 不包含音频
	// -f h264: 输出格式为原始 H.264 流
	cmd := exec.Command("ffmpeg",
		"-i", inputVideo,
		"-vcodec", "libx264", // 重新编码为 H.264
		"-preset", "medium",
		"-crf", "23",
		"-an", // 不包含音频
		"-f", "h264",
		"-y", // 覆盖已存在的文件
		outputH264,
	)

	fmt.Println("\n正在转换...")
	fmt.Println("ffmpeg 命令:", cmd.String())

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("转换失败: %v\n输出: %s", err, string(output))
	}

	// 获取文件大小
	info, err := os.Stat(outputH264)
	if err != nil {
		log.Fatalf("无法读取输出文件信息: %v", err)
	}

	fmt.Printf("\n✓ 转换成功!\n")
	fmt.Printf("输出文件: %s\n", outputH264)
	fmt.Printf("文件大小: %d 字节 (%.2f MB)\n", info.Size(), float64(info.Size())/1024/1024)

	// 获取绝对路径
	absPath, _ := filepath.Abs(outputH264)
	fmt.Printf("\n现在可以使用以下命令分析这个 H.264 流:\n")
	fmt.Printf("  cd examples/h264_stream_analysis\n")
	fmt.Printf("  go run main.go %s \"请描述视频内容\"\n", absPath)
}
