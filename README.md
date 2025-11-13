# 智谱 GLM-4.5V H.264 视频流分析 SDK

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.19-blue)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

用于处理 H.264/AVC 视频流并调用智谱 AI GLM-4.5V 模型进行视频内容分析的 Golang SDK。

## 特性

- ✅ H.264/AVC 视频流处理
- ✅ 自动帧提取和分辨率调整
- ✅ 支持实时视频流分析
- ✅ 简单易用的 API

## 安装

**安装依赖：**

```bash
# macOS
brew install ffmpeg

# Ubuntu/Debian
sudo apt install ffmpeg
```

**安装 SDK：**

```bash
go get github.com/t8y2/zhipu-video-sdk
```

## 快速开始

1. **设置 API Key**

```bash
export ZHIPU_API_KEY=your_api_key_here
```

2. **运行示例**

```bash
cd examples
go run main.go test.h264 "请描述视频中发生了什么"
```

3. **转换视频为 H.264（如需要）**

```bash
ffmpeg -i input.mp4 -vcodec libx264 -an -f h264 output.h264
```

## 使用示例

```go
package main

import (
    "log"
    "os"
    "github.com/t8y2/zhipu-video-sdk/client"
)

func main() {
    c := client.NewClient("")
    c.ConfigureStreamProcessor(2, 1120, 1120, 90)
    defer c.CleanupStreamProcessor()

    h264Data, _ := os.ReadFile("video.h264")
    resp, err := c.AnalyzeH264Stream(h264Data, "描述这个视频")
    if err != nil {
        log.Fatal(err)
    }

    println(resp.Choices[0].Message.Content)
}
```

## API

### 主要方法

- `NewClient(apiKey string)` - 创建客户端
- `AnalyzeH264Stream(h264Data []byte, prompt string)` - 分析 H.264 视频流
- `ConfigureStreamProcessor(fps, width, height, quality int)` - 配置处理参数
- `CleanupStreamProcessor()` - 清理临时文件

## 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。
