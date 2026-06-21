package ua

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"sharelink/internal/db"
)

// ParseKeywords converts a JSON array string of keywords into a string slice
func ParseKeywords(jsonStr string) ([]string, error) {
	if jsonStr == "" {
		return []string{}, nil
	}
	var keywords []string
	err := json.Unmarshal([]byte(jsonStr), &keywords)
	return keywords, err
}

func MatchKeywords(ua string, keywords []string, caseSensitive bool, matchType string) bool {
	if len(keywords) == 0 {
		return false
	}

	for _, kw := range keywords {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			continue
		}

		if matchType == "regex" {
			pattern := kw
			if !caseSensitive {
				pattern = "(?i)" + pattern
			}
			re, err := regexp.Compile(pattern)
			if err == nil && re.MatchString(ua) {
				return true
			}
		} else {
			// default: contains
			matchUA := ua
			matchKW := kw
			if !caseSensitive {
				matchUA = strings.ToLower(ua)
				matchKW = strings.ToLower(kw)
			}
			if strings.Contains(matchUA, matchKW) {
				return true
			}
		}
	}
	return false
}

// ValidateUA checks if the User-Agent is allowed by the policy
func ValidateUA(userAgent string, policy *db.UAPolicy) (bool, string) {
	if policy == nil || !policy.Enabled || policy.Mode == "disabled" {
		return true, ""
	}

	// 1. Check for empty UA
	if userAgent == "" {
		if policy.AllowEmptyUA {
			return true, ""
		}
		return false, "empty_ua_blocked"
	}

	blockKeywords, _ := ParseKeywords(policy.BlockKeywords)
	allowKeywords, _ := ParseKeywords(policy.AllowKeywords)

	// 2. Check blacklist first (blacklist has priority)
	if policy.Mode == "blacklist" || policy.Mode == "mixed" {
		if MatchKeywords(userAgent, blockKeywords, policy.CaseSensitive, policy.MatchType) {
			return false, "ua_blocked"
		}
	}

	// 3. Check whitelist
	if policy.Mode == "whitelist" || policy.Mode == "mixed" {
		if len(allowKeywords) > 0 {
			if !MatchKeywords(userAgent, allowKeywords, policy.CaseSensitive, policy.MatchType) {
				return false, "ua_blocked"
			}
		}
	}

	return true, ""
}

// GetGlobalUAPolicy returns the currently active global User-Agent policy
func GetGlobalUAPolicy() *db.UAPolicy {
	setting, found, err := db.FindGlobalSetting("global_ua_policy_id")
	if err != nil || !found {
		return nil
	}

	policyID, err := strconv.ParseUint(setting.Value, 10, 64)
	if err != nil {
		return nil
	}

	var policy db.UAPolicy
	err = db.DB.First(&policy, "id = ?", policyID).Error
	if err != nil {
		return nil
	}

	return &policy
}
