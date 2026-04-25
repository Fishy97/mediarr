package probe

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Result struct {
	DurationSeconds float64  `json:"durationSeconds,omitempty"`
	VideoCodec      string   `json:"videoCodec,omitempty"`
	AudioCodecs     []string `json:"audioCodecs,omitempty"`
	SubtitleCodecs  []string `json:"subtitleCodecs,omitempty"`
	Width           int      `json:"width,omitempty"`
	Height          int      `json:"height,omitempty"`
	Source          string   `json:"source"`
	Warning         string   `json:"warning,omitempty"`
}

type ffprobePayload struct {
	Streams []struct {
		CodecType string `json:"codec_type"`
		CodecName string `json:"codec_name"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

func File(ctx context.Context, path string) Result {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", path)
	output, err := cmd.Output()
	if err != nil {
		return inferFromName(path, err)
	}

	var payload ffprobePayload
	if err := json.Unmarshal(output, &payload); err != nil {
		return inferFromName(path, err)
	}

	result := Result{Source: "ffprobe"}
	if payload.Format.Duration != "" {
		result.DurationSeconds, _ = strconv.ParseFloat(payload.Format.Duration, 64)
	}
	for _, stream := range payload.Streams {
		switch stream.CodecType {
		case "video":
			if result.VideoCodec == "" {
				result.VideoCodec = stream.CodecName
				result.Width = stream.Width
				result.Height = stream.Height
			}
		case "audio":
			result.AudioCodecs = appendUnique(result.AudioCodecs, stream.CodecName)
		case "subtitle":
			result.SubtitleCodecs = appendUnique(result.SubtitleCodecs, stream.CodecName)
		}
	}
	return result
}

func inferFromName(path string, cause error) Result {
	result := Result{Source: "filename", Warning: "ffprobe unavailable: " + cause.Error()}
	lower := strings.ToLower(path)
	switch {
	case strings.Contains(lower, "2160p"):
		result.Width = 3840
		result.Height = 2160
	case strings.Contains(lower, "1080p"):
		result.Width = 1920
		result.Height = 1080
	case strings.Contains(lower, "720p"):
		result.Width = 1280
		result.Height = 720
	}
	switch {
	case strings.Contains(lower, "x265"), strings.Contains(lower, "hevc"):
		result.VideoCodec = "hevc"
	case strings.Contains(lower, "x264"):
		result.VideoCodec = "h264"
	case strings.Contains(lower, "av1"):
		result.VideoCodec = "av1"
	}
	if errors.Is(cause, exec.ErrNotFound) {
		result.Warning = "ffprobe not installed; filename-derived probe only"
	}
	return result
}

func appendUnique(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
