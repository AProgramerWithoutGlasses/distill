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
