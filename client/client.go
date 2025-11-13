package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/t8y2/zhipu-video-sdk/models"
	"github.com/t8y2/zhipu-video-sdk/processor"
)

func init() {
	// 加载 .env 配置
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")
	_ = godotenv.Load("../../.env")
}

const (
	DefaultAPIURL = "https://open.bigmodel.cn/api/paas/v4/chat/completions"
	DefaultModel  = "glm-4.5v"
	EnvAPIKey     = "ZHIPU_API_KEY"
)

// Client GLM-4.5V 客户端 (专注于 H.264/AVC 视频流处理)
type Client struct {
	APIKey          string
	APIURL          string
	Model           string
	HTTPClient      *http.Client
	StreamProcessor *processor.StreamProcessor // H.264/AVC 流处理器
}

// NewClient 创建客户端，apiKey 为空时从环境变量 ZHIPU_API_KEY 读取
func NewClient(apiKey string) *Client {
	if apiKey == "" {
		apiKey = os.Getenv(EnvAPIKey)
	}

	return &Client{
		APIKey:          apiKey,
		APIURL:          DefaultAPIURL,
		Model:           DefaultModel,
		StreamProcessor: processor.NewStreamProcessor(),
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// AnalyzeFrames 分析图像帧
// 帧应该是符合 GLM-4V 要求的 JPEG 格式：
// - 分辨率能被 28 整除（如 1120x1120）
// - 高质量编码，避免过度压缩
func (c *Client) AnalyzeFrames(prompt string, frames [][]byte) (*models.ChatResponse, error) {
	return c.AnalyzeFramesWithOptions(prompt, frames, nil)
}

// AnalyzeFramesWithOptions 使用自定义选项分析图像帧
func (c *Client) AnalyzeFramesWithOptions(prompt string, frames [][]byte, options *ChatOptions) (*models.ChatResponse, error) {
	// 构造请求内容
	contents := []models.Content{
		{
			Type: "text",
			Text: prompt,
		},
	}

	// 添加图像帧（使用 base64 编码的 data URI）
	for _, frame := range frames {
		base64Image := base64.StdEncoding.EncodeToString(frame)
		contents = append(contents, models.Content{
			Type: "image_url",
			ImageURL: &models.ImageURL{
				URL:    fmt.Sprintf("data:image/jpeg;base64,%s", base64Image),
				Detail: "high", // 使用高细节模式获得最佳分析效果
			},
		})
	}

	req := models.ChatRequest{
		Model: c.Model,
		Messages: []models.Message{
			{
				Role:    "user",
				Content: contents,
			},
		},
	}

	// 应用自定义选项
	if options != nil {
		req.Temperature = options.Temperature
		req.TopP = options.TopP
		req.MaxTokens = options.MaxTokens
		req.Stream = options.Stream
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.APIURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var chatResp models.ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &chatResp, nil
}

// ChatOptions 包含可选的对话参数
type ChatOptions struct {
	Temperature *float64 // 0.0-1.0, 控制随机性
	TopP        *float64 // 0.0-1.0, 核采样参数
	MaxTokens   *int     // 最大生成 token 数
	Stream      bool     // 是否启用流式响应
}

// AnalyzeH264Stream 分析 H.264/AVC 编码的视频流
// 这适用于实时视频流场景，类似于 glm-realtime-sdk-video 的实现
// h264Data: 原始 H.264 编码的视频数据
// prompt: 分析提示词
func (c *Client) AnalyzeH264Stream(h264Data []byte, prompt string) (*models.ChatResponse, error) {
	return c.AnalyzeH264StreamWithOptions(h264Data, prompt, nil)
}

// AnalyzeH264StreamWithOptions 使用自定义选项分析 H.264 视频流
func (c *Client) AnalyzeH264StreamWithOptions(h264Data []byte, prompt string, options *ChatOptions) (*models.ChatResponse, error) {
	// 使用 StreamProcessor 处理 H.264 流
	base64Frames, err := c.StreamProcessor.ProcessH264Stream(h264Data)
	if err != nil {
		return nil, fmt.Errorf("failed to process H.264 stream: %w", err)
	}

	// 将 base64 帧转换为字节数组
	frames := make([][]byte, len(base64Frames))
	for i, b64Frame := range base64Frames {
		frameData, err := base64.StdEncoding.DecodeString(b64Frame)
		if err != nil {
			return nil, fmt.Errorf("failed to decode frame %d: %w", i, err)
		}
		frames[i] = frameData
	}

	return c.AnalyzeFramesWithOptions(prompt, frames, options)
}

// ConfigureStreamProcessor 配置 H.264 流处理器
// fps: 帧率（推荐 2）
// width, height: 目标分辨率（必须能被 28 整除）
// quality: JPEG 质量（1-100，推荐 85-95）
func (c *Client) ConfigureStreamProcessor(fps, width, height, quality int) {
	c.StreamProcessor.WithFPS(fps).
		WithResolution(width, height).
		WithQuality(quality)
}

// SetStreamSPSPPS 设置 H.264 流的 SPS 和 PPS 参数
// 这些参数对于正确解码 H.264 流是必需的
// 如果不设置，将使用默认值
func (c *Client) SetStreamSPSPPS(sps, pps string) {
	c.StreamProcessor.WithSPSPPS(sps, pps)
}

// CleanupStreamProcessor 清理流处理器创建的临时文件
func (c *Client) CleanupStreamProcessor() error {
	return c.StreamProcessor.Cleanup()
}
