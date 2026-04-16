# YouTube Content Discovery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 `pkg/youtube/youtube.go` 中的 `YoutubeClient`（策略 A + B），结果由 `GenContent` 通过 `s.dao.SaveDiscoveredVideos()` 持久化到 `discovered_videos` 表。

**Architecture:** 包级单例模式。`youtube.Youtubeclient` 在 `pkg/initial.go` 的 `InitPkg(app)` 中初始化，持有 API Key。`TriggerDiscoverNew` / `TriggerDiscoverClassic` 返回 `[]dao.DiscoveredVideo`，由 Service 层负责保存，职责分离。

**Tech Stack:** `google.golang.org/api/youtube/v3`，GORM `clause.OnConflict`，MySQL

---

## 文件变更总览

| 文件 | 操作 | 职责 |
|------|------|------|
| `pkg/settings/settings.go` | 修改 | 新增 `YouTubeConfig` + 挂载到 `AppConfig` |
| `configs/local.yaml` | 修改 | 修正 youtube 缩进（当前为 1 空格） |
| `dao/mysql.go` | 修改 | 自动建库，消除 "Unknown database" 错误 |
| `dao/dao.go` | 修改 | 新增 `Article` + `SaveArticle`，`DiscoveredVideo` + `SaveDiscoveredVideos`，`AutoMigrate` |
| `pkg/youtube/youtube.go` | 修改（当前为空实现） | `YoutubeClient` 完整实现：两种策略 + 预过滤 |
| `service/generate_service.go` | 修改 | `GenContent` 内调用 `youtube.Youtubeclient` 并保存结果 |

---

## Task 1：依赖安装 + 配置扩展

**Files:**
- Run: `go get google.golang.org/api/youtube/v3`
- Modify: `pkg/settings/settings.go`
- Modify: `configs/local.yaml`

- [ ] **Step 1: 安装 YouTube API 客户端**

```bash
cd "d:/GoLand 2023.3/distill"
go get google.golang.org/api/youtube/v3
```

Expected: `go.mod` / `go.sum` 更新，无报错。

- [ ] **Step 2: 在 `pkg/settings/settings.go` 中新增 `YouTubeConfig`**

在 `LLMConfig` 定义之后添加：

```go
type YouTubeConfig struct {
	APIKey string `mapstructure:"api_key"`
}
```

将 `AppConfig` 改为：

```go
type AppConfig struct {
	Mode string `mapstructure:"mode"`
	Port int    `mapstructure:"port"`

	*LogConfig     `mapstructure:"log"`
	*MySQLConfig   `mapstructure:"mysql"`
	*RedisConfig   `mapstructure:"redis"`
	*LLMConfig     `mapstructure:"llm"`
	*YouTubeConfig `mapstructure:"youtube"`
}
```

- [ ] **Step 3: 修正 `configs/local.yaml` 的 youtube 缩进**

当前文件末尾是（1 空格缩进，viper 可能解析失败）：
```yaml
youtube:
 api_key: "AIzaSyAzI_7kpR8HYEkWg-J_RX4c2YNAPumby1o"
```

改为（2 空格，与其他块一致）：
```yaml
youtube:
  api_key: "AIzaSyAzI_7kpR8HYEkWg-J_RX4c2YNAPumby1o"
```

- [ ] **Step 4: 编译检查**

```bash
cd "d:/GoLand 2023.3/distill"
go build ./...
```

Expected: 无报错。

- [ ] **Step 5: Commit**

```bash
git add pkg/settings/settings.go configs/local.yaml go.mod go.sum
git commit -m "feat: add YouTubeConfig and youtube/v3 dependency"
```

---

## Task 2：Dao 层

**Files:**
- Modify: `dao/mysql.go`
- Modify: `dao/dao.go`

- [ ] **Step 1: 替换 `dao/mysql.go`（新增自动建库）**

```go
package dao

import (
	"fmt"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"goweb_staging/pkg/settings"
)

func initDB(m *settings.MySQLConfig) *gorm.DB {
	// 先不带库名连接，自动创建目标库（避免 "Unknown database" 错误）
	rootDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local",
		m.User, m.Password, m.Host, m.Port)
	rootDB, err := gorm.Open(mysql.Open(rootDSN), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		zap.L().Error("gorm root connect failed", zap.Error(err))
		return nil
	}
	rootDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4", m.DB))

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		m.User, m.Password, m.Host, m.Port, m.DB)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		zap.L().Error("gorm init failed", zap.Error(err))
		return nil
	}
	zap.L().Info("gorm init success", zap.String("db", m.DB))
	return db
}
```

- [ ] **Step 2: 替换 `dao/dao.go`（新增两个模型及写入方法）**

```go
package dao

import (
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"goweb_staging/pkg/settings"
)

type Dao struct {
	db  *gorm.DB
	rdb *redis.Client
}

// Article 存储已生成的公众号文章
type Article struct {
	gorm.Model
	URL    string `gorm:"not null"`
	Title  string
	Intro  string `gorm:"type:text"`
	Body   string `gorm:"type:longtext"`
	Ending string `gorm:"type:text"`
}

// DiscoveredVideo 存储从 YouTube 发现的候选视频
type DiscoveredVideo struct {
	gorm.Model
	VideoID     string    `gorm:"uniqueIndex;not null"`
	ChannelID   string    `gorm:"not null"`
	Title       string    `gorm:"not null"`
	Description string    `gorm:"type:text"`
	PublishedAt time.Time
	ViewCount   int64
	Mode        string `gorm:"not null"`                    // "A" 新内容 / "B" 历史经典
	Status      string `gorm:"not null;default:'pending'"` // pending / filtered / generated
}

func Init(app *settings.AppConfig) *Dao {
	db := initDB(app.MySQLConfig)
	if db != nil {
		if err := db.AutoMigrate(&Article{}, &DiscoveredVideo{}); err != nil {
			zap.L().Error("AutoMigrate failed", zap.Error(err))
		}
	}
	return &Dao{
		db: db,
		// rdb: initRDB(app.RedisConfig),
	}
}

// SaveArticle 将生成的文章写入数据库，返回自增 ID。
func (d *Dao) SaveArticle(url, title, intro, body, ending string) (uint, error) {
	if d.db == nil {
		return 0, fmt.Errorf("数据库未初始化")
	}
	article := Article{URL: url, Title: title, Intro: intro, Body: body, Ending: ending}
	if err := d.db.Create(&article).Error; err != nil {
		return 0, err
	}
	return article.ID, nil
}

// SaveDiscoveredVideos 批量写入候选视频，video_id 重复时静默跳过。
func (d *Dao) SaveDiscoveredVideos(videos []DiscoveredVideo) error {
	if d.db == nil {
		return fmt.Errorf("数据库未初始化")
	}
	if len(videos) == 0 {
		return nil
	}
	return d.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&videos).Error
}
```

- [ ] **Step 3: 编译检查**

```bash
cd "d:/GoLand 2023.3/distill"
go build ./...
```

Expected: 无报错。

- [ ] **Step 4: Commit**

```bash
git add dao/dao.go dao/mysql.go
git commit -m "feat: add Article/DiscoveredVideo models and SaveDiscoveredVideos"
```

---

## Task 3：实现 `YoutubeClient`

**Files:**
- Modify: `pkg/youtube/youtube.go`

- [ ] **Step 1: 替换整个文件**

```go
package youtube

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/api/option"
	googleYoutube "google.golang.org/api/youtube/v3"
	"goweb_staging/dao"
	"goweb_staging/pkg/settings"
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
func (c *YoutubeClient) TriggerDiscoverNew() []dao.DiscoveredVideo {
	ctx := context.Background()
	yt, err := googleYoutube.NewService(ctx, option.WithAPIKey(c.apiKey))
	if err != nil {
		zap.L().Error("youtube.NewService failed", zap.Error(err))
		return nil
	}

	since := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	var all []dao.DiscoveredVideo

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
		videos := fetchAndFilter(yt, ids, "A", func(v *googleYoutube.Video) bool {
			return true // 策略 A 无额外数值过滤
		})
		zap.L().Info("mode A fetched", zap.String("channel", channelID), zap.Int("count", len(videos)))
		all = append(all, videos...)
	}
	return all
}

// TriggerDiscoverClassic 策略 B：拉取白名单频道播放量最高的历史视频。
func (c *YoutubeClient) TriggerDiscoverClassic() []dao.DiscoveredVideo {
	ctx := context.Background()
	yt, err := googleYoutube.NewService(ctx, option.WithAPIKey(c.apiKey))
	if err != nil {
		zap.L().Error("youtube.NewService failed", zap.Error(err))
		return nil
	}

	threshold := time.Now().AddDate(0, 0, -modeBMinAgeDays)
	var all []dao.DiscoveredVideo

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
		videos := fetchAndFilter(yt, ids, "B", func(v *googleYoutube.Video) bool {
			if int64(v.Statistics.ViewCount) < modeBMinViewCount {
				return false
			}
			publishedAt, err := time.Parse(time.RFC3339, v.Snippet.PublishedAt)
			if err != nil {
				return false
			}
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
		ids = append(ids, item.Id.VideoId)
	}
	return ids
}

// fetchAndFilter 调用 videos.list 获取完整字段，应用通用过滤（排除直播 + 标题黑名单）
// 及调用方传入的额外过滤函数。
func fetchAndFilter(
	yt *googleYoutube.Service,
	videoIDs []string,
	mode string,
	extraFilter func(*googleYoutube.Video) bool,
) []dao.DiscoveredVideo {
	detailResp, err := yt.Videos.
		List([]string{"id", "snippet", "statistics", "liveStreamingDetails"}).
		Id(strings.Join(videoIDs, ",")).
		Do()
	if err != nil {
		zap.L().Error("videos.list failed", zap.Error(err))
		return nil
	}

	var results []dao.DiscoveredVideo
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
		publishedAt, _ := time.Parse(time.RFC3339, v.Snippet.PublishedAt)
		results = append(results, dao.DiscoveredVideo{
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
```

- [ ] **Step 2: 编译检查**

```bash
cd "d:/GoLand 2023.3/distill"
go build ./...
```

Expected: 无报错。

- [ ] **Step 3: Commit**

```bash
git add pkg/youtube/youtube.go
git commit -m "feat: implement YoutubeClient with strategy A and B"
```

---

## Task 4：集成到 `GenContent`

**Files:**
- Modify: `service/generate_service.go`

- [ ] **Step 1: 在 `GenContent` 开头添加 discover 调用**

在 `youtubeTranscriptApi(req.URL)` 调用之前，插入：

```go
import (
    // 新增
    "goweb_staging/pkg/youtube"
    // 其余保持不变
)

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
	// ... 以下不变
```

- [ ] **Step 2: 编译检查**

```bash
cd "d:/GoLand 2023.3/distill"
go build ./...
```

Expected: 无报错。

- [ ] **Step 3: 启动服务，手动验证**

```bash
curl -X POST http://localhost:8080/api/gen-content \
  -H "Content-Type: application/json" \
  -d '{"url":"https://www.youtube.com/watch?v=liwCmj7wllE"}'
```

验证：
1. 响应正常返回（含 `title`/`intro`/`body`/`ending`）
2. 日志中出现 `mode A fetched` / `mode B fetched`
3. MySQL 查询确认有数据：

```sql
SELECT video_id, title, mode, status, view_count FROM discovered_videos LIMIT 20;
```

- [ ] **Step 4: Commit**

```bash
git add service/generate_service.go
git commit -m "feat: integrate YouTube discovery into GenContent"
```

---

## Self-Review

| Spec 要求 | 对应 Task |
|-----------|----------|
| 策略 A：24h，`order=date`，`videoDuration=long` | Task 3 `TriggerDiscoverNew` |
| 策略 B：历史经典，`order=viewCount`，`videos.list` 补充数据 | Task 3 `TriggerDiscoverClassic` |
| 过滤：时长≥20min | Task 3（`VideoDuration("long")`） |
| 过滤：排除直播回放 | Task 3 `fetchAndFilter`（`LiveStreamingDetails != nil`） |
| 过滤：标题黑名单 | Task 3 `containsBlacklistedKeyword` |
| 过滤 B：播放量≥50万 | Task 3 `extraFilter`（`ViewCount < modeBMinViewCount`） |
| 过滤 B：发布时间≥30天前 | Task 3 `extraFilter`（`publishedAt.Before(threshold)`） |
| 去重：唯一索引 + ON CONFLICT DO NOTHING | Task 2 `SaveDiscoveredVideos` |
| `DiscoveredVideo` 数据模型 | Task 2 |
| `YouTubeConfig` + API Key 在配置文件 | Task 1 |
| 常量与白名单写在代码中 | Task 3 |
| 包级单例 `youtube.Youtubeclient` | Task 3 |
| `GenContent` 调用并 `s.dao.SaveDiscoveredVideos` | Task 4 |
