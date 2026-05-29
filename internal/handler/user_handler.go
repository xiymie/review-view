package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"review-view/internal/model"
	"review-view/internal/store"
)

type UserHandler struct {
	users store.UserStore
}

func NewUserHandler(users store.UserStore) *UserHandler {
	return &UserHandler{users: users}
}

func callerRole(c *gin.Context) model.UserRole {
	role, _ := c.Get("role")
	return model.UserRole(fmt.Sprintf("%v", role))
}

func (h *UserHandler) APIList(c *gin.Context) {
	users, err := h.users.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

type createUserRequest struct {
	Username string         `json:"username" binding:"required"`
	Password string         `json:"password" binding:"required"`
	Role     model.UserRole `json:"role"`
	Email    string         `json:"email"`
	Phone    string         `json:"phone"`
	Position string         `json:"position"`
	Remark   string         `json:"remark"`
}

func (h *UserHandler) APICreate(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "参数错误"})
		return
	}

	if req.Role == "" {
		req.Role = model.UserRoleNormal
	}

	// 只有超级管理员可以创建管理员及以上权限账户
	if req.Role != model.UserRoleNormal && callerRole(c) != model.UserRoleSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"message": "只有超级管理员可以创建管理员账户"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "密码处理失败"})
		return
	}

	user := &model.User{
		Username:     req.Username,
		PasswordHash: string(hash),
		Role:         req.Role,
		Email:        req.Email,
		Phone:        req.Phone,
		Position:     req.Position,
		Remark:       req.Remark,
	}
	if err := h.users.Create(user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "用户名已存在"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) APIGet(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	user, err := h.users.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "用户不存在"})
		return
	}
	c.JSON(http.StatusOK, user)
}

type updateUserRequest struct {
	Role     model.UserRole `json:"role"`
	Email    string         `json:"email"`
	Phone    string         `json:"phone"`
	Position string         `json:"position"`
	Remark   string         `json:"remark"`
	Password string         `json:"password"`
}

func (h *UserHandler) APIUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	user, err := h.users.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "用户不存在"})
		return
	}

	// 只有超级管理员可以修改管理员及以上权限的账户
	if user.Role != model.UserRoleNormal && callerRole(c) != model.UserRoleSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"message": "只有超级管理员可以修改管理员账户"})
		return
	}

	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "参数错误"})
		return
	}

	// 只有超级管理员可以提升为管理员及以上权限
	if req.Role != "" && req.Role != model.UserRoleNormal && callerRole(c) != model.UserRoleSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"message": "只有超级管理员可以设置管理员权限"})
		return
	}

	if req.Role != "" {
		user.Role = req.Role
	}
	user.Email = req.Email
	user.Phone = req.Phone
	user.Position = req.Position
	user.Remark = req.Remark

	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "密码处理失败"})
			return
		}
		user.PasswordHash = string(hash)
	}

	if err := h.users.Update(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) APIDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	uid, _ := c.Get("uid")
	selfID := int64(0)
	switch v := uid.(type) {
	case float64:
		selfID = int64(v)
	case int64:
		selfID = v
	}
	if selfID == id {
		c.JSON(http.StatusBadRequest, gin.H{"message": "不能删除自己"})
		return
	}

	user, err := h.users.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "用户不存在"})
		return
	}

	// 只有超级管理员可以删除管理员及以上权限的账户
	if user.Role != model.UserRoleNormal && callerRole(c) != model.UserRoleSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"message": "只有超级管理员可以删除管理员账户"})
		return
	}

	// 超级管理员账户不可删除
	if user.Role == model.UserRoleSuperAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"message": "超级管理员账户不可删除"})
		return
	}

	if err := h.users.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func (h *UserHandler) APIGetMe(c *gin.Context) {
	username := fmt.Sprintf("%v", c.MustGet("username"))
	user, err := h.users.GetByUsername(username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "用户不存在"})
		return
	}
	c.JSON(http.StatusOK, user)
}

type updateMeRequest struct {
	Email              string `json:"email"`
	Phone              string `json:"phone"`
	Position           string `json:"position"`
	Remark             string `json:"remark"`
	OldPassword        string `json:"old_password"`
	NewPassword        string `json:"new_password"`
	NotifyEnabled      bool   `json:"notify_enabled"`
	NotifyEmails       string `json:"notify_emails"`
	NotifyWecomWebhook string `json:"notify_wecom_webhook"`
}

func (h *UserHandler) APIUpdateMe(c *gin.Context) {
	username := fmt.Sprintf("%v", c.MustGet("username"))
	user, err := h.users.GetByUsername(username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "用户不存在"})
		return
	}

	var req updateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "参数错误"})
		return
	}

	user.Email = req.Email
	user.Phone = req.Phone
	user.Position = req.Position
	user.Remark = req.Remark
	user.NotifyEnabled = req.NotifyEnabled
	user.NotifyEmails = req.NotifyEmails
	user.NotifyWecomWebhook = req.NotifyWecomWebhook

	if req.NewPassword != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "原密码错误"})
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "密码处理失败"})
			return
		}
		user.PasswordHash = string(hash)
	}

	if err := h.users.Update(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, user)
}
