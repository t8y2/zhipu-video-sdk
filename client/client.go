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

// Client GLM-4.5V 客户端
type Client struct {
	APIKey         string
	APIURL         string
	Model          string
	HTTPClient     *http.Client
	VideoProcessor *processor.VideoProcessor
	DefaultFPS     int
}

// NewClient 创建客户端，apiKey 为空时从环境变量 ZHIPU_API_KEY 读取
func NewClient(apiKey string) *Client {
	if apiKey == "" {
		apiKey = os.Getenv(EnvAPIKey)
	}

	return &Client{
		APIKey:         apiKey,
		APIURL:         DefaultAPIURL,
		Model:          DefaultModel,
		DefaultFPS:     2,
		VideoProcessor: processor.NewVideoProcessor(2),
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// SetFPS 调整提取帧率
func (c *Client) SetFPS(fps int) {
	c.DefaultFPS = fps
	c.VideoProcessor = processor.NewVideoProcessor(fps)
}

// AnalyzeFrames 分析图像帧
func (c *Client) AnalyzeFrames(prompt string, frames [][]byte) (*models.ChatResponse, error) {
	// 构造请求内容
	contents := []models.Content{
		{
			Type: "text",
			Text: prompt,
		},
	}

	// 添加图像帧
	for _, frame := range frames {
		base64Image := base64.StdEncoding.EncodeToString(frame)
		contents = append(contents, models.Content{
			Type: "image_url",
			ImageURL: &models.ImageURL{
				URL: fmt.Sprintf("data:image/jpeg;base64,%s", base64Image),
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

// AnalyzeVideo 分析视频内容
func (c *Client) AnalyzeVideo(videoPath string, prompt string) (*models.ChatResponse, error) {
	frames, err := c.VideoProcessor.ExtractFrames(videoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract frames: %w", err)
	}

	return c.AnalyzeFrames(prompt, frames)
}

// ConfigureProcessor 配置视频处理器性能参数
func (c *Client) ConfigureProcessor(maxWorkers, bufferSize int, enableHWAccel bool) {
	c.VideoProcessor.WithMaxWorkers(maxWorkers).
		WithBufferSize(bufferSize).
		WithHWAccel(enableHWAccel)
}

// AnalyzeVideoFromStream 从内存中的视频数据分析内容
func (c *Client) AnalyzeVideoFromStream(videoData []byte, prompt string) (*models.ChatResponse, error) {
	frames, err := c.VideoProcessor.ExtractFramesFromStream(videoData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract frames from stream: %w", err)
	}

	return c.AnalyzeFrames(prompt, frames)
}
