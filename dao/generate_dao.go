package dao

import (
	"fmt"
	"goweb_staging/model"
)

// SaveArticle 将生成的文章写入数据库，返回自增 ID。
func (d *Dao) SaveArticle(url, title, intro, body, ending string) (uint, error) {
	if d.db == nil {
		return 0, fmt.Errorf("数据库未初始化")
	}
	article := model.Article{
		URL:    url,
		Title:  title,
		Intro:  intro,
		Body:   body,
		Ending: ending,
	}
	if err := d.db.Create(&article).Error; err != nil {
		return 0, err
	}
	return article.ID, nil
}
