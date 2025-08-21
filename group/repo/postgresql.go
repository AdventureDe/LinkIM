package repo

import (
	"log"

	"github.com/AdventureDe/tempName/group/repo/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// InitDB 初始化数据库连接
func InitDB() (*gorm.DB, error) {
	dsn := "user=hassin password=12345678 dbname=project2 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	DB = db

	// 自动迁移
	autoMigrate()

	return DB, nil
}

// autoMigrate 自动迁移所有模型
func autoMigrate() {
	err := DB.AutoMigrate(
		&model.Group{},
	)
	if err != nil {
		log.Fatal("数据库迁移失败：", err)
	}
}

// CloseDB 关闭数据库连接
func CloseDB() {
	sqlDB, err := DB.DB() // 获取底层的 *sql.DB
	if err != nil {
		log.Println("获取 sql.DB 实例失败：", err)
		return
	}
	err = sqlDB.Close() // 关闭连接池
	if err != nil {
		log.Println("关闭数据库连接失败：", err)
	}
}
