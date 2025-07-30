package api

import (
	"center/model"
	"center/pkg/db"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type GrantRequest struct {
	AppIdList  []string `json:"appIdList"`  // ID列表
	UserIdList []string `json:"userIdList"` // 用户ID列表
	SyncFlag   int      `json:"syncFlag"`   // 同步标志
}

func GrantUsers(r *http.Request, body []byte) error {
	// 检查Content-Type
	if !strings.Contains(r.Header.Get("Content-Type"), contentTypeJSON) {
		return fmt.Errorf("Content-Type must be %s", contentTypeJSON)
	}

	// 解析JSON到结构体
	var req GrantRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return fmt.Errorf("grantUsers failed to parse JSON: %w", err)
	}

	// 处理同步逻辑
	return buildGrantUserApps(req)
}

// buildGrantUserApps processes the user and app lists and returns the userApps slice or an error.
func buildGrantUserApps(req GrantRequest) error {
	for _, userID := range req.UserIdList {
		id, err := strconv.ParseUint(userID, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid user ID %s: %w", userID, err)
		}
		user, err := db.DB.GetUserByID(id)
		if err != nil {
			return err
		}
		userApps, err := buildAppsForUser(user, req.AppIdList)
		if err != nil {
			return err
		}
		err = handleSync(userApps)
		if err != nil {
			return fmt.Errorf("failed to handle sync for user %s: %w", user.UserName, err)
		}
	}
	return nil
}

// buildAppsForUser builds ProxyUserApp entries for a user and a list of app IDs.
func buildAppsForUser(user model.XjrUser, appIdList []string) ([]model.ProxyUserApp, error) {
	var proxyUserApps []model.ProxyUserApp
	for _, appIDStr := range appIdList {
		id, err := strconv.ParseUint(appIDStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid app ID %s: %w", appIDStr, err)
		}

		// 获取应用的发布地址
		publishAddressInside, err := getAddressesByID(id)
		if err != nil {
			return nil, fmt.Errorf("failed to get publish address for app ID %d: %w", id, err)
		}
		// 获取app_id
		appID, err := db.DB.GetJosAppIDByID(id)
		if err != nil {
			return nil, fmt.Errorf("failed to get app ID for id %d: %w", id, err)
		}
		// 将用户信息和应用发布地址同步到sqlite数据库
		var proxyUserApp model.ProxyUserApp
		proxyUserApp.UserName = user.UserName
		proxyUserApp.Name = user.Name
		proxyUserApp.Mobile = user.Mobile
		proxyUserApp.Email = user.Email
		proxyUserApp.AppID = appID
		proxyUserApp.AppAddress = publishAddressInside

		proxyUserApps = append(proxyUserApps, proxyUserApp)
	}
	return proxyUserApps, nil
}
