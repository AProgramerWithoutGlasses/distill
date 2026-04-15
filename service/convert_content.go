package service

import (
	"bytes"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

// extractVideoID 从 YouTube URL 中提取 video ID。
// 支持格式：
//   - https://www.youtube.com/watch?v=liwCmj7wllE
//   - https://youtu.be/liwCmj7wllE
func extractVideoID(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("URL 解析失败: %w", err)
	}
	// youtu.be 短链：video ID 在 path 里
	if u.Host == "youtu.be" {
		id := strings.TrimPrefix(u.Path, "/")
		if id == "" {
			return "", fmt.Errorf("无效的 YouTube 短链: %s", rawURL)
		}
		return id, nil
	}
	// 标准链：video ID 在 query 参数 v 里
	id := u.Query().Get("v")
	if id == "" {
		return "", fmt.Errorf("URL 中未找到 video ID: %s", rawURL)
	}
	return id, nil
}

func youtubeTranscriptApi(videoURL string) (string, error) {
	// 将videoUrl转换为videoID
	//  https://www.youtube.com/watch?v=liwCmj7wllE --> liwCmj7wllE
	videoID, err := extractVideoID(videoURL)
	if err != nil {
		return "", err
	}


	cmd := exec.Command("youtube_transcript_api",
		videoID,
		"--languages", "zh-Hans", "zh-Hant", "en",
		//"--http-proxy", "http://127.0.0.1:7890",
		//"--https-proxy", "http://127.0.0.1:7890",
	)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("命令执行失败: %v, 原因: %s", err, stderr.String())
	}

	return out.String(), nil
}

// parseTranscript 从 youtube_transcript_api 的原始输出中提取纯文本。
// 原始格式形如：[{'text': 'hello', 'start': 0.0, ...}, {'text': "don't", ...}]
// text 值可能用单引号或双引号，取决于内容是否含撇号。
func parseTranscript(raw string) string {
	var parts []string
	marker := "'text': "
	i := 0
	for {
		idx := strings.Index(raw[i:], marker)
		if idx == -1 {
			break
		}
		i += idx + len(marker)
		if i >= len(raw) {
			break
		}
		quote := raw[i] // ' 或 "
		i++
		end := strings.IndexByte(raw[i:], quote)
		if end == -1 {
			break
		}
		parts = append(parts, raw[i:i+end])
		i += end + 1
	}
	return strings.Join(parts, " ")
}
