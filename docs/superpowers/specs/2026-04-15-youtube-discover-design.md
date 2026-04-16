# YouTube 内容发现模块设计文档

**日期**：2026-04-15
**阶段**：Phase 2 — 内容自动化拉取（策略 A + B）
**范围**：`pkg/youtube.go`（新建）、`dao/dao.go`（新表）、`service/service.go`（集成）、配置扩展

---

## 1. 背景与目标

Phase 1 已跑通"手动输入 YouTube URL → 生成文章"的核心链路。Phase 2 第一步是**自动发现候选内容**，替代人工输入 URL。

目标：从预设的 YouTube 频道白名单中，按两种策略自动拉取视频，经机械预过滤后写入数据库，供后续质量筛选（Phase 2 第二步）消费。

---

## 2. 两种拉取策略

### 策略 A — 追踪新内容

- 拉取白名单频道在**过去 24 小时**内发布的视频
- API：`search.list`，参数 `publishedAfter=24h前`, `order=date`, `type=video`, `videoDuration=long`
- 目的：保持时效性，捕捉大佬的最新发声

### 策略 B — 挖掘历史经典

- 拉取白名单频道**播放量最高**的历史视频（每频道最多 `modeBMaxResults` 条）
- API：`search.list`，参数 `order=viewCount`, `type=video`, `videoDuration=long`
- 再调 `videos.list` 补充 `viewCount` 和 `publishedAt` 的精确值
- 目的：挖掘已被时间检验的优质内容，不受发布时间限制

---

## 3. 机械预过滤规则（无 AI）

以下规则在写入数据库**之前**执行，纯规则判断：

| 规则 | 适用策略 | 实现方式 |
|------|---------|---------|
| 时长 ≥ 20 分钟 | A + B | API 参数 `videoDuration=long` |
| 排除直播回放 | A + B | 检查 `videos.list` 返回的 `liveStreamingDetails` 字段为空 |
| 标题黑名单关键词 | A + B | 本地字符串匹配（不区分大小写）：`clip`, `shorts`, `highlights`, `compilation`, `trailer`, `reacts to`, `reaction` |
| 播放量 ≥ 50 万次 | 仅 B | 从 `videos.list` 取 `viewCount` 后过滤 |
| 发布时间 ≥ 30 天前 | 仅 B | 确保是"历史经典"，排除近期视频 |
| video_id 去重 | A + B | 数据库 `video_id` 唯一索引，重复时用 `ON CONFLICT DO NOTHING` 跳过 |

---

## 4. 架构设计

### 4.1 数据模型 — `discovered_videos` 表

```go
type DiscoveredVideo struct {
    gorm.Model
    VideoID     string    `gorm:"uniqueIndex;not null"`
    ChannelID   string    `gorm:"not null"`
    Title       string    `gorm:"not null"`
    Description string    `gorm:"type:text"`
    PublishedAt time.Time
    ViewCount   int64
    Mode        string    `gorm:"not null"` // "A" 或 "B"
    Status      string    `gorm:"not null;default:'pending'"` // pending / filtered / generated
}
```

`Status` 字段作为各阶段之间的流水线状态机：
- `pending`：已发现，等待质量筛选
- `filtered`：质量筛选通过，等待生成文章
- `generated`：文章已生成

### 4.2 `YouTubeDiscoverer` 结构体

文件：`pkg/youtube.go`

```go
type YouTubeDiscoverer struct{}

func (d *YouTubeDiscoverer) TriggerDiscoverNew()
func (d *YouTubeDiscoverer) TriggerDiscoverClassic()
```

### 4.3 常量

```go
const (
    modeBMaxResults     = 10
    modeBMinViewCount   = 500_000
    modeBMinAgeDays     = 30
)

var whitelistedChannels = []string{
    "UCSHZKyawb77ixDdsGog4iWA", // Lex Fridman
    "UCWX3yGbODI3HMCTBPbLDEgw", // Andrej Karpathy
    // 后续按需添加
}

var titleBlacklist = []string{
    "clip", "shorts", "#shorts", "highlights",
    "compilation", "trailer", "reacts to", "reaction",
}
```

### 4.4 内部调用流程

**DiscoverNew：**
```
for each channel:
    search.list(publishedAfter=24h, order=date, videoDuration=long, type=video)
    → 对每条结果调 videos.list 获取完整字段（liveStreamingDetails, viewCount）
    → 过滤：排除直播回放 + 标题黑名单
    → 批量 upsert（ON CONFLICT video_id DO NOTHING）
```

**DiscoverClassic：**
```
for each channel:
    search.list(order=viewCount, videoDuration=long, type=video, maxResults=modeBMaxResults)
    → 调 videos.list 补充 viewCount + publishedAt
    → 过滤：排除直播回放 + 标题黑名单 + viewCount<50万 + 发布时间<30天
    → 批量 upsert（ON CONFLICT video_id DO NOTHING）
```

### 4.5 Service 集成

`Service` 结构体不新增字段。`YouTubeDiscoverer` 在 `GenContent` 内作为局部变量使用：

```go
func (s *Service) GenContent(req GenContentReq) (*GenContentResp, error) {
    var youTubeDiscoverer YouTubeDiscoverer
    youTubeDiscoverer.TriggerDiscoverNew()
    youTubeDiscoverer.TriggerDiscoverClassic()
    // ... 后续字幕抓取 → 生成文章流程不变
}
```

### 4.6 调用方式

`TriggerDiscoverNew` 和 `TriggerDiscoverClassic` 是 `YouTubeDiscoverer` 上的两个独立方法，
在 `GenContent` 调用链内触发，不暴露为独立 HTTP 接口。

---

## 5. 配置扩展

`configs/local.yaml` 新增：
```yaml
youtube:
  api_key: "AIza..."
```.

`pkg/settings/settings.go` 新增：
```go
type YouTubeConfig struct {
    APIKey string `mapstructure:"api_key"`
}
```

`AppConfig` 新增 `*YouTubeConfig` 字段。

---

## 6. 依赖

YouTube Data API 官方 Go 客户端：
```
google.golang.org/api/youtube/v3
```

---

## 7. 不在本次范围内

- 定时触发（cron/APScheduler）：Phase 2 后期再加
- 质量筛选（filter_content.go）：下一个独立模块
- 其他平台（Twitter、博客 RSS）：后续扩展
