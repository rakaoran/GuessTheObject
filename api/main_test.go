package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestServerSecurity(t *testing.T) {
	r := CreateServer([]string{"http://localhost:3000", "https://oussama.com"})
	r.GET("/testroute", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "success")
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()

	r.ServeHTTP(res, req)

	assert.Equal(t, bytes.NewBufferString("healthy"), res.Body)

	req = httptest.NewRequest(http.MethodGet, "/testroute", nil)
	req.Header.Add("Origin", "http://evil.com")
	res = httptest.NewRecorder()

	r.ServeHTTP(res, req)
	assert.Equal(t, http.StatusForbidden, res.Code)
	assert.Equal(t, "forbidden origin", res.Body.String())

	req = httptest.NewRequest(http.MethodGet, "/testroute", nil)
	req.Header.Add("Origin", "https://oussama.com")
	res = httptest.NewRecorder()

	r.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "success", res.Body.String())

}
