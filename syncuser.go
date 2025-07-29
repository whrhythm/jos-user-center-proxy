package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

type UserRequest struct {
	UserName        string   `json:"userName"`
	ConfirmPassword string   `json:"confirmPassword"`
	DepartmentID    string   `json:"departmentId"`
	Name            string   `json:"name"`
	Mobile          string   `json:"mobile"`
	Email           string   `json:"email"`
	AppIDList       []string `json:"appIdList"` // 重点字段1
	SyncFlag        int      `json:"syncFlag"`  // 重点字段2
	CheckFlag       int      `json:"checkFlag"`
	Password        string   `json:"password"`
}

func syncUser(r *http.Request, path string) error {

	// 检查Content-Type
	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		return fmt.Errorf("Content-Type must be application/json")
	}
	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}

	// 解析JSON到结构体
	var req UserRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	if req.SyncFlag == 1 {
		for _, appIDStr := range req.AppIDList {
			appID, err := strconv.ParseUint(appIDStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid appId %s: %w", appIDStr, err)
			}

			// 获取应用的发布地址
			publishAddressInside, publishAddressOutside, err := getAddressesByAppID(appID)
			if err != nil {
				return fmt.Errorf("failed to get addresses for appId %d: %w", appID, err)
			}

			// 这里可以添加同步用户到应用的逻辑
			fmt.Printf("Syncing user for appId %d with inside address %s and outside address %s\n",
				appID, publishAddressInside, publishAddressOutside)
		}
	}
	fmt.Print("Syncing user data for path: ", path)
	return nil
}

func getAddressesByAppID(appID uint64) (string, string, error) {
	var app JosApp
	result := DB.Select("publish_address_inside", "publish_address_outside").
		Where("id = ?", appID).
		First(&app)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return "", "", fmt.Errorf("application with ID %d not found", appID)
		}
		return "", "", fmt.Errorf("database error: %v", result.Error)
	}

	return app.PublishAddressInside, app.PublishAddressOutside, nil
}
