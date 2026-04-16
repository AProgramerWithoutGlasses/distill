package model

import (
	"time"

	"gorm.io/gorm"
)

// DiscoveredVideo 存储从 YouTube 发现的候选视频
type DiscoveredVideo struct {
	gorm.Model
	VideoID     string    `gorm:"uniqueIndex;not null;size:255"`
	ChannelID   string    `gorm:"not null"`
	Title       string    `gorm:"not null"`
	Description string    `gorm:"type:text"`
	PublishedAt time.Time
	ViewCount   int64
	Mode        string `gorm:"not null"`                   // "A" 新内容 / "B" 历史经典
	Status      string `gorm:"not null;default:'pending'"` // pending / filtered / generated
}
