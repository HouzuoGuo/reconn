package httpsvc

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/re-connect-ai/reconn/db/dbgen"
)

// handleCreateUser is a gin handler that creates a new user.
func (svc *HttpService) handleCreateUser(c *gin.Context) {
	var req dbgen.CreateUserParams
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	user, err := svc.Database.CreateUser(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, user)
}

// handleListUsers is a gin handler that lists all users.
func (svc *HttpService) handleListUsers(c *gin.Context) {
	users, err := svc.Database.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, users)
}
