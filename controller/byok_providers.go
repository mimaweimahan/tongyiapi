package controller

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetByokProvidersAdmin lists the full BYOK vendor catalog (enabled + disabled).
// Root only. This configures what users can attach personal keys to — not shared channels.
func GetByokProvidersAdmin(c *gin.Context) {
	rows, err := model.ListAllByokProviders()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"items": rows,
		"note":  "此处配置用户自带钥的厂商目录，不是站长共享渠道",
	})
}

// CreateByokProviderAdmin adds a vendor to the user-facing /my-keys catalog.
func CreateByokProviderAdmin(c *gin.Context) {
	var req model.ByokProviderInput
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	row, err := model.CreateByokProvider(req)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, row)
}

// UpdateByokProviderAdmin edits an existing catalog entry (id immutable).
func UpdateByokProviderAdmin(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	var req model.ByokProviderInput
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	row, err := model.UpdateByokProvider(id, req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "provider not found")
			return
		}
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, row)
}

type byokProviderEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

// SetByokProviderEnabledAdmin toggles whether the vendor appears on /my-keys.
func SetByokProviderEnabledAdmin(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	var req byokProviderEnabledRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	row, err := model.SetByokProviderEnabled(id, req.Enabled)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "provider not found")
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, row)
}

type byokProviderSortRequest struct {
	IDs []string `json:"ids"`
}

// ReorderByokProvidersAdmin sets sort order from a full or partial id list.
func ReorderByokProvidersAdmin(c *gin.Context) {
	var req byokProviderSortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.ReorderByokProviders(req.IDs); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	rows, err := model.ListAllByokProviders()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": rows})
}
