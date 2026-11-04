package controller

import (
	"errors"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetByokEntitlementsAdmin lists BYOK provider entitlements (Root).
func GetByokEntitlementsAdmin(c *gin.Context) {
	userId, _ := strconv.Atoi(strings.TrimSpace(c.Query("user_id")))
	providerId := strings.TrimSpace(c.Query("provider_id"))
	status := strings.TrimSpace(c.Query("status"))
	username := strings.TrimSpace(c.Query("username"))
	if userId <= 0 && username != "" {
		var user model.User
		if err := model.DB.Where("username = ?", username).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				common.ApiErrorMsg(c, "user not found")
				return
			}
			common.ApiError(c, err)
			return
		}
		userId = user.Id
	}
	rows, err := model.ListByokEntitlements(userId, providerId, status)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"items": rows,
		"note":  "开通后用户仍自带官方 API Key；本站不代持、不转卖共享钥",
	})
}

type grantByokEntitlementRequest struct {
	UserId     int    `json:"user_id"`
	Username   string `json:"username"`
	ProviderId string `json:"provider_id"`
	Note       string `json:"note"`
}

// GrantByokEntitlementAdmin activates access to a paid BYOK vendor for a user.
func GrantByokEntitlementAdmin(c *gin.Context) {
	var req grantByokEntitlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	userId := req.UserId
	if userId <= 0 && strings.TrimSpace(req.Username) != "" {
		var user model.User
		if err := model.DB.Where("username = ?", strings.TrimSpace(req.Username)).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				common.ApiErrorMsg(c, "user not found")
				return
			}
			common.ApiError(c, err)
			return
		}
		userId = user.Id
	}
	grantedBy := c.GetInt("id")
	row, err := model.GrantByokEntitlement(userId, req.ProviderId, grantedBy, req.Note)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, row)
}

type revokeByokEntitlementRequest struct {
	UserId     int    `json:"user_id"`
	Username   string `json:"username"`
	ProviderId string `json:"provider_id"`
}

// RevokeByokEntitlementAdmin revokes paid-vendor access (user keeps free vendors).
func RevokeByokEntitlementAdmin(c *gin.Context) {
	id, _ := strconv.Atoi(strings.TrimSpace(c.Param("id")))
	if id > 0 {
		row, err := model.RevokeByokEntitlementById(id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				common.ApiErrorMsg(c, "entitlement not found")
				return
			}
			common.ApiErrorMsg(c, err.Error())
			return
		}
		common.ApiSuccess(c, row)
		return
	}
	var req revokeByokEntitlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	userId := req.UserId
	if userId <= 0 && strings.TrimSpace(req.Username) != "" {
		var user model.User
		if err := model.DB.Where("username = ?", strings.TrimSpace(req.Username)).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				common.ApiErrorMsg(c, "user not found")
				return
			}
			common.ApiError(c, err)
			return
		}
		userId = user.Id
	}
	row, err := model.RevokeByokEntitlement(userId, req.ProviderId)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, row)
}
