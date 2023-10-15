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
	user, err := svc.Config.Database.CreateUser(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, user)
}

// handleListUsers is a gin handler that lists all users.
func (svc *HttpService) handleListUsers(c *gin.Context) {
	users, err := svc.Config.Database.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, users)
}

// handleGetUserByName is a gin handler that retrieves a user by name.
func (svc *HttpService) handleGetUserByName(c *gin.Context) {
	user, err := svc.Config.Database.GetUserByName(c.Request.Context(), c.Params.ByName("user_name"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, user)
}