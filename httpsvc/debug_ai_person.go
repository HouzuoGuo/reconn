package httpsvc

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/re-connect-ai/reconn/db/dbgen"
)

// handleCreateAIPerson a gin handler that creates a AI personality with its voice model and context prompt.
func (svc *HttpService) handleCreateAIPerson(c *gin.Context) {
	var req dbgen.CreateAIPersonParams
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	aiPerson, err := svc.Config.Database.CreateAIPerson(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, aiPerson)
}

// handleListUsers is a gin handler that lists all AI personality for a user.
func (svc *HttpService) handleListAIPersons(c *gin.Context) {
	userID, _ := strconv.Atoi(c.Params.ByName("user_name"))
	aiPersons, err := svc.Config.Database.ListAIPersons(c.Request.Context(), int64(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, aiPersons)
}

// handleGetUserByName is a gin handler that retrieves a user by name.
func (svc *HttpService) handleUpdateAIPerson(c *gin.Context) {
	aiPersonID, _ := strconv.Atoi(c.Params.ByName("ai_person_id"))
	var req dbgen.UpdateAIPersonContextPromptByIDParams
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	req.ID = int64(aiPersonID)
	err := svc.Config.Database.UpdateAIPersonContextPromptByID(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}
