package models

// Message represents a chat message
type Message struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

// Content represents message content (text or image)
type Content struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL represents an image URL
// Supports both URL and base64 encoded images
// For optimal GLM-4V performance:
// - Use base64 encoded JPEG images
// - Resolution should be divisible by 28 (e.g., 1120x1120)
// - Recommended quality: 85-95
type ImageURL struct {
	URL    string `json:"url"`              // URL or data URI (data:image/jpeg;base64,...)
	Detail string `json:"detail,omitempty"` // Optional: "auto", "low", "high"
}

// ChatRequest represents the API request
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature *float64  `json:"temperature,omitempty"` // Optional: 0.0-1.0
	TopP        *float64  `json:"top_p,omitempty"`       // Optional: 0.0-1.0
	MaxTokens   *int      `json:"max_tokens,omitempty"`  // Optional: max tokens to generate
	Stream      bool      `json:"stream,omitempty"`      // Optional: enable streaming
}

// ChatResponse represents the API response
type ChatResponse struct {
	ID      string `json:"id"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// FrameMetadata contains metadata about extracted video frames
type FrameMetadata struct {
	TotalFrames    int     `json:"total_frames"`
	FPS            int     `json:"fps"`
	Duration       float64 `json:"duration"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	ValidDimension bool    `json:"valid_dimension"` // Whether dimensions meet requirements
}
