package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	ByokPricingFree = "free"
	ByokPricingPaid = "paid"

	ByokEntitlementActive  = "active"
	ByokEntitlementRevoked = "revoked"
)

// ByokProviderEntitlement records that a user may connect a paid BYOK vendor
// with their own official key (scheme B). Not a shared-key handout.
type ByokProviderEntitlement struct {
	Id         int    `json:"id" gorm:"primaryKey"`
	UserId     int    `json:"user_id" gorm:"column:user_id;not null;uniqueIndex:uk_byok_ent_user_provider;index"`
	ProviderId string `json:"provider_id" gorm:"column:provider_id;size:64;not null;uniqueIndex:uk_byok_ent_user_provider;index"`
	Status     string `json:"status" gorm:"size:16;not null;default:active;index"`
	GrantedBy  int    `json:"granted_by" gorm:"not null;default:0"`
	GrantedAt  int64  `json:"granted_at" gorm:"bigint;not null;default:0"`
	RevokedAt  int64  `json:"revoked_at" gorm:"bigint;not null;default:0"`
	Note       string `json:"note" gorm:"size:255"`
}

func (ByokProviderEntitlement) TableName() string { return "byok_provider_entitlements" }

func NormalizeByokPricingTier(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case ByokPricingPaid:
		return ByokPricingPaid
	default:
		return ByokPricingFree
	}
}

func CanConnectByokProvider(userId int, pricingTier, providerId string) bool {
	if NormalizeByokPricingTier(pricingTier) != ByokPricingPaid {
		return true
	}
	return HasActiveByokEntitlement(userId, providerId)
}

func HasActiveByokEntitlement(userId int, providerId string) bool {
	if userId <= 0 || strings.TrimSpace(providerId) == "" || DB == nil {
		return false
	}
	var count int64
	err := DB.Model(&ByokProviderEntitlement{}).
		Where("user_id = ? AND provider_id = ? AND status = ?", userId, strings.TrimSpace(providerId), ByokEntitlementActive).
		Count(&count).Error
	return err == nil && count > 0
}

func ListByokEntitlements(userId int, providerId, status string) ([]ByokProviderEntitlement, error) {
	if DB == nil {
		return nil, errors.New("db not ready")
	}
	q := DB.Model(&ByokProviderEntitlement{}).Order("id desc")
	if userId > 0 {
		q = q.Where("user_id = ?", userId)
	}
	if pid := strings.TrimSpace(providerId); pid != "" {
		q = q.Where("provider_id = ?", pid)
	}
	if st := strings.TrimSpace(strings.ToLower(status)); st != "" {
		q = q.Where("status = ?", st)
	}
	var rows []ByokProviderEntitlement
	if err := q.Limit(500).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func GrantByokEntitlement(userId int, providerId string, grantedBy int, note string) (*ByokProviderEntitlement, error) {
	if DB == nil {
		return nil, errors.New("db not ready")
	}
	if userId <= 0 {
		return nil, errors.New("user_id is required")
	}
	providerId = strings.TrimSpace(providerId)
	if !byokProviderIDRe.MatchString(providerId) {
		return nil, errors.New("invalid provider id")
	}
	if _, err := GetByokProvider(providerId); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("provider not found")
		}
		return nil, err
	}
	var user User
	if err := DB.Select("id").Where("id = ?", userId).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	note = strings.TrimSpace(note)
	if len([]rune(note)) > 200 {
		return nil, errors.New("note too long")
	}
	now := time.Now().Unix()
	var existing ByokProviderEntitlement
	err := DB.Where("user_id = ? AND provider_id = ?", userId, providerId).First(&existing).Error
	if err == nil {
		existing.Status = ByokEntitlementActive
		existing.GrantedBy = grantedBy
		existing.GrantedAt = now
		existing.RevokedAt = 0
		existing.Note = note
		if saveErr := DB.Save(&existing).Error; saveErr != nil {
			return nil, saveErr
		}
		return &existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	row := ByokProviderEntitlement{
		UserId:     userId,
		ProviderId: providerId,
		Status:     ByokEntitlementActive,
		GrantedBy:  grantedBy,
		GrantedAt:  now,
		Note:       note,
	}
	if createErr := DB.Create(&row).Error; createErr != nil {
		return nil, createErr
	}
	return &row, nil
}

func RevokeByokEntitlement(userId int, providerId string) (*ByokProviderEntitlement, error) {
	if DB == nil {
		return nil, errors.New("db not ready")
	}
	if userId <= 0 {
		return nil, errors.New("user_id is required")
	}
	providerId = strings.TrimSpace(providerId)
	if providerId == "" {
		return nil, errors.New("provider_id is required")
	}
	var existing ByokProviderEntitlement
	err := DB.Where("user_id = ? AND provider_id = ?", userId, providerId).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("entitlement not found")
		}
		return nil, err
	}
	if existing.Status == ByokEntitlementRevoked {
		return &existing, nil
	}
	existing.Status = ByokEntitlementRevoked
	existing.RevokedAt = time.Now().Unix()
	if err := DB.Save(&existing).Error; err != nil {
		return nil, err
	}
	return &existing, nil
}

func RevokeByokEntitlementById(id int) (*ByokProviderEntitlement, error) {
	if DB == nil {
		return nil, errors.New("db not ready")
	}
	if id <= 0 {
		return nil, errors.New("id is required")
	}
	var existing ByokProviderEntitlement
	if err := DB.Where("id = ?", id).First(&existing).Error; err != nil {
		return nil, err
	}
	if existing.Status == ByokEntitlementRevoked {
		return &existing, nil
	}
	existing.Status = ByokEntitlementRevoked
	existing.RevokedAt = time.Now().Unix()
	if err := DB.Save(&existing).Error; err != nil {
		return nil, err
	}
	return &existing, nil
}
