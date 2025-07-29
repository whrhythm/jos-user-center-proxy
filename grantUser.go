package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

type GrantRequest struct {
	AppIdList  []string `json:"appIdList"`  // 应用ID列表
	UserIdList []string `json:"userIdList"` // 用户ID列表
	SyncFlag   int      `json:"syncFlag"`   // 同步标志
}

func grantUsers(r *http.Request, body []byte) error {
	// 检查Content-Type
	if !strings.Contains(r.Header.Get("Content-Type"), contentTypeJSON) {
		return fmt.Errorf("Content-Type must be %s", contentTypeJSON)
	}

	// 解析JSON到结构体
	var req GrantRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return fmt.Errorf("grantUsers failed to parse JSON: %w", err)
	}

	for _, userID := range req.UserIdList {
		id, user, err := validateAndFetchUser(userID)
		if err != nil {
			return err
		}
		if err := processAppIDsForUser(id, user, req.AppIdList); err != nil {
			return err
		}
	}

	return nil
}

// validateAndFetchUser parses the userID and fetches the user from the DB.
func validateAndFetchUser(userID string) (uint64, XjrUser, error) {
	id, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		return 0, XjrUser{}, fmt.Errorf("invalid user ID %s: %w", userID, err)
	}

	var user XjrUser
	if err := DB.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, XjrUser{}, fmt.Errorf("user with ID %d not found", id)
		}
		return 0, XjrUser{}, fmt.Errorf("failed to query user with ID %d: %w", id, err)
	}
	return id, user, nil
}

// processAppIDsForUser processes all app IDs for a given user.
func processAppIDsForUser(userID uint64, user XjrUser, appIdList []string) error {
	for _, appIDStr := range appIdList {
		appID, err := strconv.ParseUint(appIDStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid app ID %s: %w", appIDStr, err)
		}

		publishAddressInside, err := getAddressesByAppID(appID)
		if err != nil {
			return fmt.Errorf("failed to get publish address for app ID %d: %w", appID, err)
		}

		if err := handleSync(userID, user.UserName, user.Name, user.Mobile, user.Email, publishAddressInside); err != nil {
			return fmt.Errorf("failed to handle sync for user ID %d: %w", userID, err)
		}
	}
	return nil
}
