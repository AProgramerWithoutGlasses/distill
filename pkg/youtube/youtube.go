package youtube

import (
	"context"
	"strings"
	"time"

	"goweb_staging/model"
	"goweb_staging/pkg/settings"

	"go.uber.org/zap"
	"google.golang.org/api/option"
	googleYoutube "google.golang.org/api/youtube/v3"
)

// -------- 常量与白名单 --------

const (
	modeBMaxResults   = 10
	modeBMinViewCount = 500_000
	modeBMinAgeDays   = 30
)

var whitelistedChannels = []string{
	"UCSHZKyawb77ixDdsGog4iWA", // Lex Fridman
	"UCWX3yGbODI3HMCTBPbLDEgw", // Andrej Karpathy
}

var titleBlacklist = []string{
	"clip", "shorts", "#shorts", "highlights",
	"compilation", "trailer", "reacts to", "reaction",
}

// -------- YoutubeClient --------

var Youtubeclient YoutubeClient

type YoutubeClient struct {
	apiKey string
}

func NewYoutubeClient(app *settings.AppConfig) {
	Youtubeclient = YoutubeClient{
		apiKey: app.YouTubeConfig.APIKey,
	}
}

// TriggerDiscoverNew 策略 A：拉取白名单频道过去 24h 内的长视频。
func (c *YoutubeClient) TriggerDiscoverNew() []model.DiscoveredVideo {
	ctx := context.Background()
	yt, err := googleYoutube.NewService(ctx, option.WithAPIKey(c.apiKey))
	if err != nil {
		zap.L().Error("youtube.NewService failed", zap.Error(err))
		return nil
	}

	since := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	var all []model.DiscoveredVideo

	for _, channelID := range whitelistedChannels {
		resp, err := yt.Search.List([]string{"id"}).
			ChannelId(channelID).
			PublishedAfter(since).
			Order("date").
			Type("video").
			VideoDuration("long").
			MaxResults(50).
			Do()
		if err != nil {
			zap.L().Error("search.list failed", zap.String("channel", channelID), zap.Error(err))
			continue
		}
		if len(resp.Items) == 0 {
			continue
		}

		ids := extractIDs(resp.Items)
		if len(ids) == 0 {
			continue
		}

		// 根据搜索到的videoID去搜索详细Video
		videos := fetchAndFilter(yt, ids, "A", func(v *googleYoutube.Video) bool {
			return true
		})
		zap.L().Info("mode A fetched", zap.String("channel", channelID), zap.Int("count", len(videos)))
		all = append(all, videos...)
	}
	return all
}

// TriggerDiscoverClassic 策略 B：拉取白名单频道播放量最高的历史视频。
func (c *YoutubeClient) TriggerDiscoverClassic() []model.DiscoveredVideo {
	ctx := context.Background()
	yt, err := googleYoutube.NewService(ctx, option.WithAPIKey(c.apiKey))
	if err != nil {
		zap.L().Error("youtube.NewService failed", zap.Error(err))
		return nil
	}

	threshold := time.Now().AddDate(0, 0, -modeBMinAgeDays)
	var all []model.DiscoveredVideo

	for _, channelID := range whitelistedChannels {
		resp, err := yt.Search.List([]string{"id"}).
			ChannelId(channelID).
			Order("viewCount").
			Type("video").
			VideoDuration("long").
			MaxResults(int64(modeBMaxResults)).
			Do()
		if err != nil {
			zap.L().Error("search.list failed", zap.String("channel", channelID), zap.Error(err))
			continue
		}
		if len(resp.Items) == 0 {
			continue
		}

		ids := extractIDs(resp.Items)
		if len(ids) == 0 {
			continue
		}
		videos := fetchAndFilter(yt, ids, "B", func(v *googleYoutube.Video) bool {
			if int64(v.Statistics.ViewCount) < modeBMinViewCount {
				return false
			}
			publishedAt, err := time.Parse(time.RFC3339, v.Snippet.PublishedAt)
			if err != nil {
				return false
			}
			// 只保留 30 天前发布的视频，确保是经过时间检验的历史经典
			return publishedAt.Before(threshold)
		})
		zap.L().Info("mode B fetched", zap.String("channel", channelID), zap.Int("count", len(videos)))
		all = append(all, videos...)
	}
	return all
}

// -------- 内部辅助函数 --------

func extractIDs(items []*googleYoutube.SearchResult) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		if item.Id != nil && item.Id.VideoId != "" {
			ids = append(ids, item.Id.VideoId)
		}
	}
	return ids
}

func fetchAndFilter(
	yt *googleYoutube.Service,
	videoIDs []string,
	mode string,
	extraFilter func(*googleYoutube.Video) bool,
) []model.DiscoveredVideo {
	detailResp, err := yt.Videos.
		List([]string{"id", "snippet", "statistics", "liveStreamingDetails"}).
		Id(strings.Join(videoIDs, ",")).
		Do()
	if err != nil {
		zap.L().Error("videos.list failed", zap.Error(err))
		return nil
	}

	var results []model.DiscoveredVideo
	for _, v := range detailResp.Items {
		if v.LiveStreamingDetails != nil {
			continue
		}
		if containsBlacklistedKeyword(v.Snippet.Title) {
			continue
		}
		if !extraFilter(v) {
			continue
		}
		publishedAt, err := time.Parse(time.RFC3339, v.Snippet.PublishedAt)
		if err != nil {
			zap.L().Warn("failed to parse publishedAt, skipping video",
				zap.String("videoId", v.Id),
				zap.String("raw", v.Snippet.PublishedAt),
				zap.Error(err))
			continue
		}
		results = append(results, model.DiscoveredVideo{
			VideoID:     v.Id,
			ChannelID:   v.Snippet.ChannelId,
			Title:       v.Snippet.Title,
			Description: v.Snippet.Description,
			PublishedAt: publishedAt,
			ViewCount:   int64(v.Statistics.ViewCount),
			Mode:        mode,
			Status:      "pending",
		})
	}
	return results
}

func containsBlacklistedKeyword(title string) bool {
	lower := strings.ToLower(title)
	for _, kw := range titleBlacklist {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}
