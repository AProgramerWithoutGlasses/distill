package model

import "gorm.io/gorm"

// Article 对应数据库中的文章记录
type Article struct {
	gorm.Model
	URL    string `gorm:"not null"`
	Title  string
	Intro  string `gorm:"type:text"`
	Body   string `gorm:"type:longtext"`
	Ending string `gorm:"type:text"`
}
