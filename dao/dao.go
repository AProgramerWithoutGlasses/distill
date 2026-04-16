package dao

import (
	"fmt"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"goweb_staging/model"
	"goweb_staging/pkg/settings"
)

type Dao struct {
	db  *gorm.DB
	rdb *redis.Client
}

func Init(app *settings.AppConfig) *Dao {
	db := initDB(app.MySQLConfig)
	if db != nil {
		if err := db.AutoMigrate(&model.Article{}, &model.DiscoveredVideo{}); err != nil {
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
	article := model.Article{URL: url, Title: title, Intro: intro, Body: body, Ending: ending}
	if err := d.db.Create(&article).Error; err != nil {
		return 0, err
	}
	return article.ID, nil
}

// SaveDiscoveredVideos 批量写入候选视频，video_id 重复时静默跳过。
func (d *Dao) SaveDiscoveredVideos(videos []model.DiscoveredVideo) error {
	if d.db == nil {
		return fmt.Errorf("数据库未初始化")
	}
	if len(videos) == 0 {
		return nil
	}
	return d.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&videos).Error
}
