# 智谱 GLM-4.5V 视频分析 SDK

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.19-blue)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

智谱 GLM-4.5V 视频分析 SDK 是一个用于调用智谱 AI 的 GLM-4.5V 多模态大模型进行视频内容分析的 Golang SDK。

## 目录结构

```
.
├── README.md                    # 项目说明文档
├── go.mod                       # Go 模块定义
├── client/                      # API 客户端
│   └── client.go               # GLM-4.5V API 客户端实现
├── models/                      # 数据模型
│   └── types.go                # 请求和响应数据结构
├── processor/                   # 视频处理器
│   └── video_processor.go      # 视频帧提取功能
└── examples/                    # 示例代码
    └── main.go                 # 基础使用示例
```

## 快速开始

### 1. 环境准备

#### 安装 Go

确保已安装 Go 1.19 或更高版本：

```bash
go version
```

#### 安装 FFmpeg

视频帧提取依赖 FFmpeg 和 FFprobe：

**macOS:**

```bash
brew install ffmpeg
```

**Ubuntu/Debian:**

```bash
sudo apt update
sudo apt install ffmpeg
```

**其他系统**: 请访问 [FFmpeg 官网](https://ffmpeg.org/download.html) 下载安装

### 2. 安装 SDK

```bash
go get github.com/t8y2/zhipu-video-sdk
```

### 3. 配置 API Key

您需要设置 `ZHIPU_API_KEY` 环境变量。可以通过以下两种方式之一进行设置：

#### 方式一：直接设置环境变量

```bash
export ZHIPU_API_KEY=your_api_key_here
```

#### 方式二：使用 .env 文件

复制环境变量示例文件并修改：

```bash
cp .env.example .env
```

然后编辑 `.env` 文件，填入您的 API 密钥：

```
ZHIPU_API_KEY=your_api_key_here
```

> 注：API 密钥可在 [智谱 AI 开放平台](https://open.bigmodel.cn/) 注册开发者账号后创建获取

### 4. 运行示例

```bash
cd examples
go run main.go
```

## 使用说明

### 基础用法

```go
package main

import (
    "fmt"
    "log"

    "github.com/t8y2/zhipu-video-sdk/client"
)

func main() {
    // 初始化客户端（从 ZHIPU_API_KEY 环境变量读取）
    vlmClient := client.NewClient("")

    // 分析视频内容（默认每秒提取 2 帧）
    videoPath := "path/to/video.mp4"
    prompt := "请分析这个视频中的内容"

    response, err := vlmClient.AnalyzeVideo(videoPath, prompt)
    // response, err := vlmClient.AnalyzeVideoFromStream(videoData, "请分析这个视频")
    if err != nil {
        log.Fatal(err)
    }

    // 打印结果
    fmt.Println(response.Choices[0].Message.Content)
}
```

### 自定义配置

```go
// 自定义 API Key
vlmClient := client.NewClient("your_custom_api_key")

// 自定义模型
vlmClient.Model = "glm-4.5v"

// 自定义帧率（每秒提取 5 帧）
vlmClient.SetFPS(5)
```

## API 文档

### Client

#### `NewClient(apiKey string) *Client`

创建新的 GLM-4.5V API 客户端。

- `apiKey`: API 密钥，如果为空则从 `ZHIPU_API_KEY` 环境变量读取
- 返回: 配置好的客户端实例，默认 FPS 为 2

#### `(*Client) AnalyzeVideo(videoPath string, prompt string) (*models.ChatResponse, error)`

分析视频文件内容（推荐使用）。

- `videoPath`: 视频文件路径
- `prompt`: 分析提示词
- 返回: API 响应或错误

#### `(*Client) AnalyzeVideoFromStream(videoData []byte, prompt string) (*models.ChatResponse, error)`

从内存中的视频数据分析内容。

- `videoData`: 视频的字节数据
- `prompt`: 分析提示词
- 返回: API 响应或错误

#### `(*Client) AnalyzeFrames(prompt string, frames [][]byte) (*models.ChatResponse, error)`

分析已提取的视频帧内容（用于自定义帧处理）。

- `prompt`: 分析提示词
- `frames`: 视频帧的 JPEG 字节数组
- 返回: API 响应或错误

#### `(*Client) SetFPS(fps int)`

设置视频帧提取帧率。

- `fps`: 每秒提取的帧数

### VideoProcessor

客户端内置 `VideoProcessor`，通常不需要直接使用。如需自定义帧处理，可通过 `vlmClient.VideoProcessor` 访问。

## 支持的模型

- `glm-4.5v` (默认) - GLM-4.5V 多模态模型
- 其他智谱 AI 支持的视觉模型

## 常见问题

### 1. FFmpeg 未找到

确保已正确安装 FFmpeg 并添加到系统 PATH。

### 2. API 调用失败

- 检查 API Key 是否正确
- 确认网络连接正常
- 查看错误信息中的状态码和详细说明

## 参考资料

- [智谱 AI 开放平台](https://open.bigmodel.cn/)
- [GLM-4.5V 模型文档](https://open.bigmodel.cn/dev/api)

## 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。
