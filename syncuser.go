package main

import (
	"bytes"
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

// 请求体结构
type UserSyncRequest struct {
	UserName string `json:"userName"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Sex      int    `json:"sex"`
}

// 响应体结构
type Response struct {
	Code     int    `json:"code"`
	Message  string `json:"message"`
	Data     any    `json:"data,omitempty"`
	ClientID string `json:"clientId,omitempty"`
}

const contentTypeJSON = "application/json"

func syncUser(r *http.Request, body []byte) error {

	// 检查Content-Type
	if !strings.Contains(r.Header.Get("Content-Type"), contentTypeJSON) {
		return fmt.Errorf("Content-Type must be %s", contentTypeJSON)
	}

	// 解析JSON到结构体
	var req UserRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	if req.SyncFlag == 1 {
		for _, appIDStr := range req.AppIDList {
			id, err := strconv.ParseUint(appIDStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid appId %s: %w", appIDStr, err)
			}

			// 获取应用的发布地址
			publishAddressInside, err := getAddressesByAppID(id)
			if err != nil {
				return fmt.Errorf("failed to get addresses for id %d: %w", id, err)
			}

			// 处理同步协议
			err = handleSync(req, id, publishAddressInside)
			if err != nil {
				return fmt.Errorf("failed to handle sync for id %d: %w", id, err)
			}
			// 这里可以添加同步用户到应用的逻辑
			fmt.Printf("Syncing user for id %d with inside address %s \n",
				id, publishAddressInside)
		}
	}
	return nil
}

func getAddressesByAppID(appID uint64) (string, error) {
	var app JosApp
	result := DB.Select("publish_address_inside").
		Where("id = ?", appID).
		First(&app)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("application with ID %d not found", appID)
		}
		return "", fmt.Errorf("database error: %v", result.Error)
	}

	return app.PublishAddressInside, nil
}

func handleSync(userRequest UserRequest, id uint64, appAuthEndpoint string) error {
	// 1. 准备请求数据
	user := UserSyncRequest{
		UserName: userRequest.UserName,
		Name:     userRequest.Name,
		Phone:    userRequest.Mobile,
		Email:    userRequest.Email,
		Sex:      0, // 默认性别为0
	}

	// 2. 序列化为JSON
	jsonData, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user data: %w", err)
	}
	// 3. 创建请求
	req, err := http.NewRequest("POST", appAuthEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	// 4. 设置请求头
	req.Header.Set("Content-Type", contentTypeJSON)
	req.Header.Set("Accept", contentTypeJSON)

	// 5. 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 6. 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// 7. 解析响应
	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// 8. 处理响应
	if response.Code == 1 {
		// 将client_id存储到JosApp表中
		var app JosApp
		result := DB.Model(&app).Where("id = ?", id).First(&app)
		if result.Error != nil {
			return fmt.Errorf("failed to find app with ID %d: %w", id, result.Error)
		}
		clientIDUint, err := strconv.ParseUint(response.ClientID, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to convert client_id to uint64: %w", err)
		}
		app.AppClientID = clientIDUint
		if err := DB.Save(&app).Error; err != nil {
			return fmt.Errorf("failed to update app with client_id: %w", err)
		}
		fmt.Printf("用户同步成功: %s (ClientID: %s)\n", userRequest.UserName, response.ClientID)
	} else {
		fmt.Printf("请求失败: %s (错误码: %d)\n", response.Message, response.Code)
	}

	return nil
}
