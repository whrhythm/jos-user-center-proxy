package main

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 数据库配置常量
const (
	DBHost     = "join-mysql-standalone-svc:3306"
	DBUser     = "root"
	DBPassword = "zn123$%^zn"
	DBName     = "user_center_workspace"
)

// JosApp 对应数据库表 jos_app
type JosApp struct {
	ID                    uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	AppID                 uint64    `gorm:"column:app_id;not null"`                           // 应用ID
	WorkspaceID           uint64    `gorm:"column:workspace_id;not null;index"`               // 工作空间ID（带索引）
	ProjectID             uint64    `gorm:"column:project_id;not null"`                       // 项目ID
	ProjectName           string    `gorm:"column:project_name;type:varchar(255)"`            // 项目名称
	AppName               string    `gorm:"column:app_name;type:varchar(255);not null"`       // 应用名称
	EnvID                 uint64    `gorm:"column:env_id"`                                    // 环境ID
	AppIcon               string    `gorm:"column:app_icon;type:varchar(500)"`                // 应用图标URL
	PublishAddressInside  string    `gorm:"column:publish_address_inside;type:varchar(255)"`  // 内部发布地址
	PublishAddressOutside string    `gorm:"column:publish_address_outside;type:varchar(255)"` // 外部发布地址
	Status                string    `gorm:"column:status;type:varchar(255)"`                  // 状态
	ClientID              uint64    `gorm:"column:client_id"`                                 // 客户端ID
	CreateDate            time.Time `gorm:"column:create_date;default:CURRENT_TIMESTAMP"`     // 创建时间（自动设置）
	ModifyDate            time.Time `gorm:"column:modify_date;autoUpdateTime"`                // 修改时间（自动更新）
}

// TableName 指定表名
func (JosApp) TableName() string {
	return "jos_app"
}

// 初始化数据库连接
func InitDB() (*gorm.DB, error) {
	// 构建DSN (Data Source Name)
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		DBUser, DBPassword, DBHost, DBName)

	// 连接数据库
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %v", err)
	}

	// 测试连接
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("Successfully connected to MySQL database")
	return db, nil
}
