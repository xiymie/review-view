package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"review-view/internal/model"
	"review-view/internal/service"
)

func (h *CredentialHandler) APIList(c *gin.Context) {
	var creds []model.RepoCredential
	var err error
	if isAdmin(c) {
		creds, err = h.credentials.List()
	} else {
		creds, err = h.credentials.ListByUser(callerUID(c))
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	uidSet := make(map[int64]struct{}, len(creds))
	for _, cr := range creds {
		if cr.CreatedBy != 0 {
			uidSet[cr.CreatedBy] = struct{}{}
		}
	}
	ownerIDs := make([]int64, 0, len(uidSet))
	for id := range uidSet {
		ownerIDs = append(ownerIDs, id)
	}
	usernames := buildUsernameMap(h.users, ownerIDs)

	type item struct {
		ID            int64  `json:"id"`
		Name          string `json:"name"`
		Username      string `json:"username"`
		OwnerUsername string `json:"owner_username"`
		CreatedAt     string `json:"created_at"`
	}

	result := make([]item, 0, len(creds))
	for _, cr := range creds {
		result = append(result, item{
			ID:            cr.ID,
			Name:          cr.Name,
			Username:      cr.Username,
			OwnerUsername: usernames[cr.CreatedBy],
			CreatedAt:     cr.CreatedAt.Format("2006-01-02 15:04"),
		})
	}
	c.JSON(http.StatusOK, result)
}

func (h *CredentialHandler) APIGet(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	cred, err := h.credentials.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		return
	}
	if !isAdmin(c) && cred.CreatedBy != callerUID(c) {
		c.JSON(http.StatusForbidden, gin.H{"message": "无权访问"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       cred.ID,
		"name":     cred.Name,
		"username": cred.Username,
	})
}

func (h *CredentialHandler) APICreate(c *gin.Context) {
	var req struct {
		Name     string `json:"name"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	cred, err := h.credentials.Create(service.CredentialCreateInput{
		Name:      req.Name,
		Username:  req.Username,
		Password:  req.Password,
		CreatedBy: callerUID(c),
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": cred.ID, "name": cred.Name})
}

func (h *CredentialHandler) APIUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	cred, err := h.credentials.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		return
	}
	if !isAdmin(c) && cred.CreatedBy != callerUID(c) {
		c.JSON(http.StatusForbidden, gin.H{"message": "无权操作"})
		return
	}

	var req struct {
		Name     string `json:"name"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	updated, err := h.credentials.Update(id, service.CredentialCreateInput{
		Name:     req.Name,
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": updated.ID, "name": updated.Name})
}

func (h *CredentialHandler) APIDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	cred, err := h.credentials.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		return
	}
	if !isAdmin(c) && cred.CreatedBy != callerUID(c) {
		c.JSON(http.StatusForbidden, gin.H{"message": "无权操作"})
		return
	}

	if err := h.credentials.Delete(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}
