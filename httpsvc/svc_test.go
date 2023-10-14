package httpsvc

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupRouter(t *testing.T) (svc *HttpService, router *gin.Engine) {
	t.Helper()
	svc = &HttpService{Config: Config{}}
	return svc, svc.SetupRouter()
}

func TestReadback(t *testing.T) {
	_, router := setupRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/debug/readback", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, `{"address":"","headers":{},"method":"GET","url":"/api/debug/readback"}`, w.Body.String())
}
