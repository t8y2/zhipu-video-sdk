package processor

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// StreamProcessor handles real-time H.264/AVC video stream processing
// Similar to the reference implementation in glm-realtime-sdk-video
type StreamProcessor struct {
	FPS          int    // Frames per second to extract (recommended: 2)
	TargetWidth  int    // Target frame width (default: 1120, must be divisible by 28)
	TargetHeight int    // Target frame height (default: 1120, must be divisible by 28)
	Quality      int    // JPEG quality (1-100, recommended: 85-95)
	SPS          string // H.264 SPS (Sequence Parameter Set) in base64
	PPS          string // H.264 PPS (Picture Parameter Set) in base64
	tempDir      string
	mu           sync.Mutex
}

// NewStreamProcessor creates a new stream processor
// Default settings aligned with GLM-4V requirements
func NewStreamProcessor() *StreamProcessor {
	return &StreamProcessor{
		FPS:          2,
		TargetWidth:  1120,
		TargetHeight: 1120,
		Quality:      90,
		// Default SPS/PPS from reference implementation
		SPS: "Z0LADJoFAAABMA==",
		PPS: "aM48gA==",
	}
}

// WithFPS sets the frames per second
func (sp *StreamProcessor) WithFPS(fps int) *StreamProcessor {
	sp.FPS = fps
	return sp
}

// WithResolution sets the target resolution
func (sp *StreamProcessor) WithResolution(width, height int) *StreamProcessor {
	// Validate that dimensions are divisible by 28
	if width%28 != 0 || height%28 != 0 {
		// Round to nearest multiple of 28
		width = ((width + 13) / 28) * 28
		height = ((height + 13) / 28) * 28
	}
	sp.TargetWidth = width
	sp.TargetHeight = height
	return sp
}

// WithQuality sets the JPEG quality
func (sp *StreamProcessor) WithQuality(quality int) *StreamProcessor {
	if quality < 1 {
		quality = 1
	}
	if quality > 100 {
		quality = 100
	}
	sp.Quality = quality
	return sp
}

// WithSPSPPS sets the H.264 SPS and PPS parameters
// These are required for proper H.264 stream decoding
func (sp *StreamProcessor) WithSPSPPS(sps, pps string) *StreamProcessor {
	sp.SPS = sps
	sp.PPS = pps
	return sp
}

// ProcessH264Stream processes H.264 video stream data and extracts frames
// This is the main function for handling real-time video streams
// Input: raw H.264/AVC encoded video data
// Output: array of base64-encoded JPEG frames
func (sp *StreamProcessor) ProcessH264Stream(h264Data []byte) ([]string, error) {
	return sp.ProcessH264StreamWithContext(context.Background(), h264Data)
}

// ProcessH264StreamWithContext processes H.264 stream with context support
func (sp *StreamProcessor) ProcessH264StreamWithContext(ctx context.Context, h264Data []byte) ([]string, error) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	// Create temp directory if not exists
	if sp.tempDir == "" {
		tempDir, err := os.MkdirTemp("", "h264stream-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp dir: %w", err)
		}
		sp.tempDir = tempDir
	}

	// 1. Inject SPS/PPS into H.264 stream
	fixedData, err := sp.injectSPSPPS(h264Data)
	if err != nil {
		return nil, fmt.Errorf("failed to inject SPS/PPS: %w", err)
	}

	// 2. Write H.264 data to temp file
	h264Path := filepath.Join(sp.tempDir, fmt.Sprintf("stream_%d.h264", time.Now().UnixNano()))
	if err := os.WriteFile(h264Path, fixedData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write h264 file: %w", err)
	}
	defer os.Remove(h264Path)

	// 3. Extract frames using ffmpeg
	frames, err := sp.extractFramesFromH264(ctx, h264Path)
	if err != nil {
		return nil, fmt.Errorf("failed to extract frames: %w", err)
	}

	// 4. Convert frames to base64
	base64Frames := make([]string, len(frames))
	for i, frame := range frames {
		base64Frames[i] = base64.StdEncoding.EncodeToString(frame)
	}

	return base64Frames, nil
}

// injectSPSPPS injects SPS and PPS NAL units into H.264 stream
// This is required for proper decoding of H.264 streams
func (sp *StreamProcessor) injectSPSPPS(h264Data []byte) ([]byte, error) {
	// Decode SPS and PPS from base64
	spsData, err := base64.StdEncoding.DecodeString(sp.SPS)
	if err != nil {
		return nil, fmt.Errorf("invalid SPS: %w", err)
	}

	ppsData, err := base64.StdEncoding.DecodeString(sp.PPS)
	if err != nil {
		return nil, fmt.Errorf("invalid PPS: %w", err)
	}

	// H.264 NAL unit start code
	startCode := []byte{0x00, 0x00, 0x00, 0x01}

	// Build fixed stream with SPS/PPS at the beginning
	var buf bytes.Buffer

	// Write SPS NAL unit
	buf.Write(startCode)
	buf.Write(spsData)

	// Write PPS NAL unit
	buf.Write(startCode)
	buf.Write(ppsData)

	// Write original H.264 data
	buf.Write(h264Data)

	return buf.Bytes(), nil
}

// extractFramesFromH264 uses ffmpeg to decode H.264 and extract JPEG frames
func (sp *StreamProcessor) extractFramesFromH264(ctx context.Context, h264Path string) ([][]byte, error) {
	// Convert quality to qscale
	qscale := 31 - int(float64(sp.Quality-1)/99.0*29.0)
	if qscale < 2 {
		qscale = 2
	}
	if qscale > 31 {
		qscale = 31
	}

	// Build ffmpeg command
	// Similar to reference implementation
	args := []string{
		"-f", "h264", // Input format: raw H.264
		"-i", h264Path,
		"-vf", fmt.Sprintf("fps=%d,scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2",
			sp.FPS, sp.TargetWidth, sp.TargetHeight, sp.TargetWidth, sp.TargetHeight),
		"-f", "image2pipe",
		"-vcodec", "mjpeg",
		"-q:v", fmt.Sprintf("%d", qscale),
		"-",
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg error: %w, stderr: %s", err, stderr.String())
	}

	// Split JPEG frames
	frames, err := sp.splitJPEGFrames(stdout.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to split frames: %w", err)
	}

	return frames, nil
}

// splitJPEGFrames splits concatenated JPEG data into individual frames
func (sp *StreamProcessor) splitJPEGFrames(data []byte) ([][]byte, error) {
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

// ProcessH264StreamReader processes H.264 stream from an io.Reader
// Useful for reading from network connections or pipes
func (sp *StreamProcessor) ProcessH264StreamReader(ctx context.Context, reader io.Reader) ([]string, error) {
	// Read all data from reader
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read stream: %w", err)
	}

	return sp.ProcessH264StreamWithContext(ctx, data)
}

// StreamFrameExtractor provides a continuous frame extraction interface
// for real-time video streaming scenarios
type StreamFrameExtractor struct {
	processor    *StreamProcessor
	frameChannel chan []byte
	errorChannel chan error
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// NewStreamFrameExtractor creates a new continuous stream frame extractor
func NewStreamFrameExtractor(processor *StreamProcessor) *StreamFrameExtractor {
	ctx, cancel := context.WithCancel(context.Background())
	return &StreamFrameExtractor{
		processor:    processor,
		frameChannel: make(chan []byte, 100),
		errorChannel: make(chan error, 10),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start begins processing H.264 stream chunks
func (sfe *StreamFrameExtractor) Start(streamReader io.Reader) {
	sfe.wg.Add(1)
	go func() {
		defer sfe.wg.Done()
		defer close(sfe.frameChannel)
		defer close(sfe.errorChannel)

		// Read stream in chunks
		buffer := make([]byte, 64*1024) // 64KB chunks
		for {
			select {
			case <-sfe.ctx.Done():
				return
			default:
				n, err := streamReader.Read(buffer)
				if err != nil {
					if err != io.EOF {
						sfe.errorChannel <- err
					}
					return
				}

				if n > 0 {
					// Process this chunk
					frames, err := sfe.processor.ProcessH264StreamWithContext(sfe.ctx, buffer[:n])
					if err != nil {
						sfe.errorChannel <- err
						continue
					}

					// Send frames to channel
					for _, frameB64 := range frames {
						frameData, _ := base64.StdEncoding.DecodeString(frameB64)
						select {
						case sfe.frameChannel <- frameData:
						case <-sfe.ctx.Done():
							return
						}
					}
				}
			}
		}
	}()
}

// GetFrameChannel returns the channel for receiving extracted frames
func (sfe *StreamFrameExtractor) GetFrameChannel() <-chan []byte {
	return sfe.frameChannel
}

// GetErrorChannel returns the channel for receiving errors
func (sfe *StreamFrameExtractor) GetErrorChannel() <-chan error {
	return sfe.errorChannel
}

// Stop stops the frame extraction
func (sfe *StreamFrameExtractor) Stop() {
	sfe.cancel()
	sfe.wg.Wait()
}

// Cleanup removes temporary files created during processing
func (sp *StreamProcessor) Cleanup() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if sp.tempDir != "" {
		err := os.RemoveAll(sp.tempDir)
		sp.tempDir = ""
		return err
	}
	return nil
}
