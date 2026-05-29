package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"review-view/internal/service"
	"review-view/internal/store"
)

type CredentialHandler struct {
	credentials *service.RepoCredentialService
	users       store.UserStore
}

func NewCredentialHandler(credentials *service.RepoCredentialService, users store.UserStore) *CredentialHandler {
	return &CredentialHandler{credentials: credentials, users: users}
}

func (h *CredentialHandler) Index(c *gin.Context) {
	creds, err := h.credentials.List()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.HTML(http.StatusOK, "credentials/index", gin.H{
		"Title":        "仓库凭据",
		"Active":       "credentials",
		"PageTemplate": "credentials/index_content",
		"Credentials":  creds,
	})
}

func (h *CredentialHandler) New(c *gin.Context) {
	c.HTML(http.StatusOK, "credentials/new", gin.H{
		"Title":        "新建凭据",
		"Active":       "credentials",
		"PageTemplate": "credentials/new_content",
		"Breadcrumbs": []breadcrumb{
			{Label: "仓库凭据", Href: "/credentials"},
			{Label: "新建凭据"},
		},
	})
}

func (h *CredentialHandler) Create(c *gin.Context) {
	_, err := h.credentials.Create(service.CredentialCreateInput{
		Name:     c.PostForm("name"),
		Username: c.PostForm("username"),
		Password: c.PostForm("password"),
	})
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(http.StatusSeeOther, "/credentials")
}

func (h *CredentialHandler) Edit(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	cred, err := h.credentials.Get(id)
	if err != nil {
		c.String(http.StatusNotFound, err.Error())
		return
	}

	c.HTML(http.StatusOK, "credentials/edit", gin.H{
		"Title":        "编辑凭据",
		"Active":       "credentials",
		"PageTemplate": "credentials/edit_content",
		"Credential":   cred,
		"Breadcrumbs": []breadcrumb{
			{Label: "仓库凭据", Href: "/credentials"},
			{Label: cred.Name},
		},
	})
}

func (h *CredentialHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	_, err := h.credentials.Update(id, service.CredentialCreateInput{
		Name:     c.PostForm("name"),
		Username: c.PostForm("username"),
		Password: c.PostForm("password"),
	})
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(http.StatusSeeOther, "/credentials")
}

func (h *CredentialHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	if err := h.credentials.Delete(id); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(http.StatusSeeOther, "/credentials")
}
