package api

import (
	"bytes"
	"center/model"
	"center/pkg/db"
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
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    []AppUserData `json:"data"`
}

type AppUserData struct {
	UserName string `json:"userName"`
	UserID   string `json:"userId"`
}

const contentTypeJSON = "application/json"

func SyncUser(r *http.Request, body []byte) error {
	// 检查Content-Type
	if !strings.Contains(r.Header.Get("Content-Type"), contentTypeJSON) {
		return fmt.Errorf("Content-Type must be %s", contentTypeJSON)
	}

	// 解析JSON到结构体
	var req UserRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// 这里同步应用，可能是在新建用户的时候，也有可能是在编辑用户
	if req.SyncFlag == 1 {
		userApps, err := buildUserApps(req)
		if err != nil {
			return err
		}
		err = handleSync(userApps)
		if err != nil {
			return fmt.Errorf("failed to handle sync %w", err)
		}
	}
	return nil
}

// buildUserApps extracts the logic for building userApps from the request.
func buildUserApps(req UserRequest) ([]model.ProxyUserApp, error) {
	var userApps []model.ProxyUserApp
	for _, appIDStr := range req.AppIDList {
		// 字符串转换为uint64
		id, err := strconv.ParseUint(appIDStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid appId %s: %w", appIDStr, err)
		}

		// 获取应用的发布地址
		publishAddressInside, err := getAddressesByID(id)
		if len(publishAddressInside) == 0 || err != nil {
			return nil, fmt.Errorf("failed to get addresses for id %d: %w", id, err)
		}

		// 获取app_id
		appID, err := db.DB.GetJosAppIDByID(id)
		if err != nil {
			return nil, fmt.Errorf("failed to get app ID for id %d: %w", id, err)
		}
		// 将用户信息和应用发布地址同步到sqlite数据库
		var userApp model.ProxyUserApp
		userApp.UserName = req.UserName
		userApp.Name = req.Name
		userApp.Mobile = req.Mobile
		userApp.Email = req.Email
		userApp.AppID = appID
		userApp.AppAddress = publishAddressInside

		userApps = append(userApps, userApp)
	}
	return userApps, nil
}

func getAddressesByID(appID uint64) (string, error) {
	var app model.JosApp
	result := db.DB.JosDb.Select("publish_address_inside").
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

func handleSync(userApps []model.ProxyUserApp) error {
	for index, app := range userApps {
		user := UserSyncRequest{
			UserName: app.UserName,
			Name:     app.Name,
			Phone:    app.Mobile,
			Email:    app.Email,
			Sex:      0,
		}

		response, err := sendUserSyncRequest(app.AppAddress, user)
		if err != nil {
			return err
		}

		if response.Code == 1 && len(response.Data) > 0 {
			fmt.Printf("请求成功: %s\n", response.Message)
			userID, err := strconv.ParseUint(response.Data[0].UserID, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse userId '%s' to uint64: %w", response.Data[0].UserID, err)
			}
			userApps[index].AppUserID = userID
		} else {
			fmt.Printf("请求失败: %s (错误码: %d)\n", response.Message, response.Code)
		}
	}

	return db.DB.UpdateProxyUser(userApps)
}

func sendUserSyncRequest(appAddress string, user UserSyncRequest) (*Response, error) {
	jsonData, err := json.Marshal(user)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user data: %w", err)
	}

	req, err := http.NewRequest("POST", appAddress, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", contentTypeJSON)
	req.Header.Set("Accept", contentTypeJSON)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}
