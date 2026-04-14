package dao

import (
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
	"goweb_staging/pkg/settings"
)

type Dao struct {
	db  *gorm.DB
	rdb *redis.Client
}

func Init(app *settings.AppConfig) *Dao {
	dao := &Dao{
		db:  initDB(app.MySQLConfig),
		rdb: initRDB(app.RedisConfig),
	}
	return dao
}
