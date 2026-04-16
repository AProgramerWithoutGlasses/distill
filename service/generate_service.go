package service

import (
	"context"
	"encoding/json"
	"fmt"
	"goweb_staging/pkg/llm"
	"goweb_staging/pkg/youtube"
	"strings"

	"go.uber.org/zap"
)

type GenContentReq struct {
	URL string `json:"url" form:"url"`
}

type GenContentResp struct {
	ContentID string `json:"content_id"`
	Title     string `json:"title"`
	Intro     string `json:"intro"`
	Body      string `json:"body"`
	Ending    string `json:"ending"`
}

// llmOutput 对应 LLM 返回的 JSON 结构
type llmOutput struct {
	Title  string `json:"title"`
	Intro  string `json:"intro"`
	Body   string `json:"body"`
	Ending string `json:"ending"`
}

func (s *Service) GenContent(req GenContentReq) (*GenContentResp, error) {
	// 发现候选内容（策略 A + B），失败不阻断主流程
	if newVideos := youtube.Youtubeclient.TriggerDiscoverNew(); len(newVideos) > 0 {
		if err := s.dao.SaveDiscoveredVideos(newVideos); err != nil {
			zap.L().Error("SaveDiscoveredVideos (A) failed", zap.Error(err))
		}
	}
	if classicVideos := youtube.Youtubeclient.TriggerDiscoverClassic(); len(classicVideos) > 0 {
		if err := s.dao.SaveDiscoveredVideos(classicVideos); err != nil {
			zap.L().Error("SaveDiscoveredVideos (B) failed", zap.Error(err))
		}
	}

	// 抓取字幕
	raw, err := youtubeTranscriptApi(req.URL)
	if err != nil {
		zap.L().Error("youtubeTranscriptApi() err", zap.Error(err))
		return nil, fmt.Errorf("抓取字幕失败: %w", err)
	}
	zap.L().Info("抓取字幕成功" + raw)

	// 提取纯文本
	transcript := parseTranscript(raw)
	zap.L().Info("提取文本成功" + transcript)

	// 文章生成
	rawOutput, err := llm.LLMclient.Chat(context.Background(), buildPrompt(transcript))
	if err != nil {
		zap.L().Error("s.llm.Chat() err", zap.Error(err))
		return nil, fmt.Errorf("文章生成失败: %w", err)
	}
	zap.L().Info("生成文章成功" + rawOutput)

	// 解析 LLM 输出
	out, err := parseLLMOutput(rawOutput)
	if err != nil {
		zap.L().Error("parseLLMOutput() err", zap.Error(err), zap.String("raw", rawOutput))
		return nil, fmt.Errorf("解析 LLM 输出失败: %w", err)
	}

	// 存储文章
	id, err := s.dao.SaveArticle(req.URL, out.Title, out.Intro, out.Body, out.Ending)
	if err != nil {
		zap.L().Error("dao.SaveArticle() err", zap.Error(err))
		return nil, fmt.Errorf("存储文章失败: %w", err)
	}

	return &GenContentResp{
		ContentID: fmt.Sprintf("%d", id),
		Title:     out.Title,
		Intro:     out.Intro,
		Body:      out.Body,
		Ending:    out.Ending,
	}, nil
}

// parseLLMOutput 从 LLM 输出中提取 JSON，兼容 markdown 代码块包裹的情况。
func parseLLMOutput(raw string) (*llmOutput, error) {
	raw = strings.TrimSpace(raw)

	// 兼容 ```json ... ``` 包裹
	if strings.HasPrefix(raw, "```") {
		newline := strings.Index(raw, "\n")
		if newline != -1 {
			raw = raw[newline+1:]
		}
		raw = strings.TrimSuffix(strings.TrimSpace(raw), "```")
		raw = strings.TrimSpace(raw)
	}

	var out llmOutput
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("json.Unmarshal failed: %w", err)
	}
	return &out, nil
}

func buildPrompt(transcript string) string {
	return fmt.Sprintf(`你是一个专业的内容创作者，擅长将英文长视频/播客的字幕内容提炼为高质量的中文微信公众号文章。

以下是视频的字幕文字稿：
<transcript>
%s
</transcript>

请将上述内容整理为一篇微信公众号文章，严格按照以下 JSON 格式输出，不要输出任何其他内容：

{
  "title": "文章标题，提炼核心信息，简洁有力，不超过20字",
  "intro": "导语，2-3句话说明这篇文章讲什么、为什么值得读，100字以内",
  "body": "正文，包含：背景铺垫（说明者是谁、在什么场合发表观点）、核心观点展开（逐一呈现关键论点，可引用原文重要表述并翻译为中文）、对中国读者的启发",
  "ending": "结尾，一句话总结全文 + 一个引导读者思考的问题"
}

要求：
- 输出纯 JSON，不要加 markdown 代码块
- 忠实呈现原始内容，不过度演绎
- 语言流畅自然，符合中文阅读习惯
- body 总长度控制在1500-3000字`, transcript)
}
