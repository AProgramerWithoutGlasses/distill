package service

import (
	"context"
	"fmt"
	"go.uber.org/zap"
)

type GenContentReq struct {
	URL string `json:"url" form:"url"`
}

type GenContentResp struct {
	ContentID string
	Title     string
	Blocks    []Block
}

type Block struct {
	Type    string
	Content string
	Caption string
}



func (s *Service) GenContent(req GenContentReq) (string, error) {
	// 抓取字幕
	raw, err := youtubeTranscriptApi(req.URL)
	if err != nil {
		zap.L().Error("YoutubeTranscriptApi() err", zap.Error(err))
		return "", fmt.Errorf("抓取字幕失败: %w", err)
	}
	zap.L().Info("抓取字幕成功" + raw)

	// 提取纯文本
	transcript := parseTranscript(raw)
	zap.L().Info("提取文本成功" + transcript)

	// 文章生成
	article, err := s.llm.Chat(context.Background(), buildPrompt(transcript))
	if err != nil {
		zap.L().Error("s.llm.Chat() err", zap.Error(err))
		return "", fmt.Errorf("文章生成失败: %w", err)
	}

	return article, nil
}



func buildPrompt(transcript string) string {
	return fmt.Sprintf(`你是一个专业的内容创作者，擅长将英文长视频/播客的字幕内容提炼为高质量的中文微信公众号文章。

以下是视频的字幕文字稿：
<transcript>
%s
</transcript>

请将上述内容整理为一篇微信公众号文章，严格按照以下结构输出：

【标题】
提炼核心信息，简洁有力，不超过20字。

【导语】
2-3句话说明这篇文章讲什么、为什么值得读，100字以内。

【正文】
- 背景铺垫：说明说话者是谁、在什么场合发表了这些观点
- 核心观点展开：逐一呈现关键论点，可适当引用原文中的重要表述（翻译为中文）
- 对中国读者的启发：这些内容对我们有什么参考价值

【结尾】
一句话总结全文 + 一个引导读者思考的问题。

要求：
- 忠实呈现原始内容，不过度演绎
- 语言流畅自然，符合中文阅读习惯
- 总长度控制在1500-3000字`, transcript)
}


