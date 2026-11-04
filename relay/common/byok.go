package common

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
)

// IsPersonalByokChannel reports whether the selected channel is a /my-keys
// personal vendor channel: group == u{id} and name == u{id}-{provider}.
// Format must stay aligned with model.PersonalGroupName / byokChannelName.
func IsPersonalByokChannel(userId int, usingGroup, channelName string) bool {
	if userId <= 0 {
		return false
	}
	personal := fmt.Sprintf("u%d", userId)
	if strings.TrimSpace(usingGroup) != personal {
		return false
	}
	prefix := personal + "-"
	name := strings.TrimSpace(channelName)
	if !strings.HasPrefix(name, prefix) {
		return false
	}
	return strings.TrimSpace(strings.TrimPrefix(name, prefix)) != ""
}

// ApplyPersonalByokFlag marks SkipSiteWallet when the request uses a personal
// BYOK channel so billing can skip users.quota / used_quota.
func ApplyPersonalByokFlag(c *gin.Context, info *RelayInfo) {
	if info == nil || info.SkipSiteWallet {
		return
	}
	channelName := ""
	if c != nil {
		channelName = common.GetContextKeyString(c, constant.ContextKeyChannelName)
		if channelName == "" {
			channelName = c.GetString("channel_name")
		}
	}
	if IsPersonalByokChannel(info.UserId, info.UsingGroup, channelName) {
		info.SkipSiteWallet = true
	}
}
