package handler

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"review-view/internal/model"
	"review-view/internal/store"
)

type breadcrumb struct {
	Label string
	Href  string
}

func parseOptionalInt(raw string) *int {
	if raw == "" {
		return nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return nil
	}
	return &value
}

func callerUID(c *gin.Context) int64 {
	uid, _ := c.Get("uid")
	switch v := uid.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	}
	return 0
}

func isAdmin(c *gin.Context) bool {
	role, _ := c.Get("role")
	r := model.UserRole(fmt.Sprintf("%v", role))
	return r == model.UserRoleAdmin || r == model.UserRoleSuperAdmin
}

// buildUsernameMap 批量将用户 ID 解析为 username，忽略不存在的 ID
func buildUsernameMap(users store.UserStore, ids []int64) map[int64]string {
	seen := make(map[int64]struct{}, len(ids))
	m := make(map[int64]string, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if u, err := users.GetByID(id); err == nil {
			m[id] = u.Username
		}
	}
	return m
}
