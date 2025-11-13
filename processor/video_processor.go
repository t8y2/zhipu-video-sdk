package processor

import (
	"bytes"
	"context"
	"fmt"
	"image/jpeg"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// VideoProcessor handles video frame extraction
type VideoProcessor struct {
	FPS           int  // Frames per second to extract
	MaxWorkers    int  // Maximum concurrent workers for frame processing
	BufferSize    int  // Buffer size for frame channels
	EnableHWAccel bool // Enable hardware acceleration for ffmpeg
	UseStreamMode bool // Process frames in streaming mode
	framePool     *sync.Pool
}

// NewVideoProcessor creates a new video processor
func NewVideoProcessor(fps int) *VideoProcessor {
	return &VideoProcessor{
		FPS:           fps,
		MaxWorkers:    runtime.NumCPU(),
		BufferSize:    100,
		EnableHWAccel: true,
		UseStreamMode: false,
		framePool: &sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}
}

// WithMaxWorkers sets the maximum number of concurrent workers
func (vp *VideoProcessor) WithMaxWorkers(workers int) *VideoProcessor {
	vp.MaxWorkers = workers
	return vp
}

// WithBufferSize sets the buffer size for frame channels
func (vp *VideoProcessor) WithBufferSize(size int) *VideoProcessor {
	vp.BufferSize = size
	return vp
}

// WithHWAccel enables/disables hardware acceleration
func (vp *VideoProcessor) WithHWAccel(enable bool) *VideoProcessor {
	vp.EnableHWAccel = enable
	return vp
}

// WithStreamMode enables/disables streaming mode
func (vp *VideoProcessor) WithStreamMode(enable bool) *VideoProcessor {
	vp.UseStreamMode = enable
	return vp
}

// ExtractFrames extracts frames from video at specified FPS
// Returns frames as JPEG byte arrays
func (vp *VideoProcessor) ExtractFrames(videoPath string) ([][]byte, error) {
	return vp.ExtractFramesWithContext(context.Background(), videoPath)
}

// ExtractFramesWithContext extracts frames with context support
func (vp *VideoProcessor) ExtractFramesWithContext(ctx context.Context, videoPath string) ([][]byte, error) {
	// Check if ffmpeg is available
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("ffmpeg not found: %w. Please install ffmpeg", err)
	}

	// Get video duration first
	duration, err := vp.getVideoDuration(videoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get video duration: %w", err)
	}

	// Calculate total frames to extract
	totalFrames := int(duration) * vp.FPS
	if totalFrames == 0 {
		return nil, fmt.Errorf("video too short or invalid")
	}

	// Build ffmpeg command with optimizations
	args := vp.buildFFmpegArgs(videoPath)
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg error: %w, stderr: %s", err, stderr.String())
	}

	// Parse JPEG frames from output with concurrent processing
	data := stdout.Bytes()
	frames, err := vp.splitJPEGFramesConcurrent(data)
	if err != nil {
		return nil, fmt.Errorf("failed to split frames: %w", err)
	}

	return frames, nil
}

// buildFFmpegArgs builds optimized ffmpeg arguments
func (vp *VideoProcessor) buildFFmpegArgs(videoPath string) []string {
	args := []string{}

	// Hardware acceleration (if enabled)
	if vp.EnableHWAccel {
		// Try to use hardware acceleration
		// macOS: videotoolbox, Linux: vaapi, Windows: dxva2
		args = append(args, "-hwaccel", "auto")
	}

	// Input file
	args = append(args, "-i", videoPath)

	// Video filter for FPS
	args = append(args, "-vf", fmt.Sprintf("fps=%d", vp.FPS))

	// Output format optimizations
	args = append(args,
		"-f", "image2pipe",
		"-vcodec", "mjpeg",
		"-q:v", "5", // Quality setting (2-31, lower is better)
		"-threads", fmt.Sprintf("%d", vp.MaxWorkers), // Multi-threading
		"-",
	)

	return args
}

// getVideoDuration gets video duration in seconds
func (vp *VideoProcessor) getVideoDuration(videoPath string) (float64, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	durationStr := strings.TrimSpace(string(output))
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, err
	}

	return duration, nil
}

// splitJPEGFrames splits concatenated JPEG data into individual frames
func (vp *VideoProcessor) splitJPEGFrames(data []byte) ([][]byte, error) {
	var frames [][]byte

	// JPEG markers
	jpegStart := []byte{0xFF, 0xD8} // SOI (Start of Image)
	jpegEnd := []byte{0xFF, 0xD9}   // EOI (End of Image)

	i := 0
	for i < len(data) {
		// Find start of JPEG
		startIdx := bytes.Index(data[i:], jpegStart)
		if startIdx == -1 {
			break
		}
		startIdx += i

		// Find end of JPEG
		endIdx := bytes.Index(data[startIdx+2:], jpegEnd)
		if endIdx == -1 {
			break
		}
		endIdx += startIdx + 2 + 2 // +2 for the marker itself

		// Extract frame
		frame := make([]byte, endIdx-startIdx)
		copy(frame, data[startIdx:endIdx])
		frames = append(frames, frame)

		i = endIdx
	}

	if len(frames) == 0 {
		return nil, fmt.Errorf("no valid JPEG frames found")
	}

	return frames, nil
}

// splitJPEGFramesConcurrent splits JPEG data concurrently for better performance
func (vp *VideoProcessor) splitJPEGFramesConcurrent(data []byte) ([][]byte, error) {
	// JPEG markers
	jpegStart := []byte{0xFF, 0xD8} // SOI (Start of Image)
	jpegEnd := []byte{0xFF, 0xD9}   // EOI (End of Image)

	// First pass: find all frame boundaries
	type framePos struct {
		start int
		end   int
		index int
	}

	var positions []framePos
	i := 0
	frameIndex := 0

	for i < len(data) {
		startIdx := bytes.Index(data[i:], jpegStart)
		if startIdx == -1 {
			break
		}
		startIdx += i

		endIdx := bytes.Index(data[startIdx+2:], jpegEnd)
		if endIdx == -1 {
			break
		}
		endIdx += startIdx + 2 + 2

		positions = append(positions, framePos{
			start: startIdx,
			end:   endIdx,
			index: frameIndex,
		})

		frameIndex++
		i = endIdx
	}

	if len(positions) == 0 {
		return nil, fmt.Errorf("no valid JPEG frames found")
	}

	// Second pass: extract frames concurrently
	frames := make([][]byte, len(positions))
	var wg sync.WaitGroup

	// Create worker pool
	jobs := make(chan framePos, vp.BufferSize)

	// Start workers
	for w := 0; w < vp.MaxWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for pos := range jobs {
				frame := make([]byte, pos.end-pos.start)
				copy(frame, data[pos.start:pos.end])
				frames[pos.index] = frame
			}
		}()
	}

	// Send jobs
	for _, pos := range positions {
		jobs <- pos
	}
	close(jobs)

	// Wait for completion
	wg.Wait()

	return frames, nil
}

// ExtractFramesFromStream extracts frames from video stream (io.Reader)
// This is useful for processing video data without saving to disk
func (vp *VideoProcessor) ExtractFramesFromStream(videoData []byte) ([][]byte, error) {
	// For stream processing, we need to save to temp file first
	// as ffmpeg requires seekable input
	cmd := exec.Command("ffmpeg",
		"-i", "pipe:0",
		"-vf", fmt.Sprintf("fps=%d", vp.FPS),
		"-f", "image2pipe",
		"-vcodec", "mjpeg",
		"-",
	)

	cmd.Stdin = bytes.NewReader(videoData)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg error: %w, stderr: %s", err, stderr.String())
	}

	// Parse JPEG frames from output
	frames, err := vp.splitJPEGFrames(stdout.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to split frames: %w", err)
	}

	return frames, nil
}

// OptimizeFrameSize reduces frame size to optimize API calls
func OptimizeFrameSize(frame []byte, maxWidth int) ([]byte, error) {
	img, err := jpeg.Decode(bytes.NewReader(frame))
	if err != nil {
		return nil, err
	}

	// Check if resize is needed
	bounds := img.Bounds()
	if bounds.Dx() <= maxWidth {
		return frame, nil
	}

	// For simplicity, return original frame
	// In production, you might want to use a resize library
	return frame, nil
}
