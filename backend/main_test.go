package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCSRFProtection(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	r := CreateServer([]string{"http://localhost:3000", "https://oussama.com"})

	r.GET("/testroute", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "success")
	})

	type testCase struct {
		name           string
		method         string
		path           string
		origin         string
		expectedStatus int
		expectedBody   string
	}

	tests := []testCase{
		{
			name:           "Health check should be public",
			method:         http.MethodGet,
			path:           "/health",
			origin:         "",
			expectedStatus: http.StatusOK,
			expectedBody:   "healthyy123",
		},
		{
			name:           "Allowed origin should pass",
			method:         http.MethodGet,
			path:           "/testroute",
			origin:         "https://oussama.com",
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name:           "Disallowed origin should be forbidden",
			method:         http.MethodGet,
			path:           "/testroute",
			origin:         "http://evil.com",
			expectedStatus: http.StatusForbidden,
			expectedBody:   "forbidden origin",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.origin != "" {
				req.Header.Add("Origin", tc.origin)
			}
			res := httptest.NewRecorder()

			r.ServeHTTP(res, req)

			assert.Equal(t, tc.expectedStatus, res.Code)
			if tc.expectedBody != "" {
				assert.Equal(t, tc.expectedBody, res.Body.String())
			}
		})
	}
}

func TestCORSHeaders(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	allowedOrigins := []string{"http://localhost:3000", "https://prod.example.com"}

	tests := []struct {
		name        string
		method      string
		reqHeaders  map[string]string
		wantCode    int
		wantHeaders map[string]string
	}{
		{
			name:   "preflight request from allowed origin",
			method: http.MethodOptions,
			reqHeaders: map[string]string{
				"Origin":                        "http://localhost:3000",
				"Access-Control-Request-Method": "POST",
			},
			wantCode: http.StatusNoContent,
			wantHeaders: map[string]string{
				"Access-Control-Allow-Origin":      "http://localhost:3000",
				"Access-Control-Allow-Methods":     "GET,POST,PUT,DELETE,OPTIONS",
				"Access-Control-Allow-Credentials": "true",
			},
		},
		{
			name:   "preflight from forbidden origin",
			method: http.MethodOptions,
			reqHeaders: map[string]string{
				"Origin":                        "http://evil.com",
				"Access-Control-Request-Method": "POST",
			},
			wantCode: http.StatusForbidden,
			wantHeaders: map[string]string{
				"Access-Control-Allow-Origin": "",
			},
		},
		{
			name:       "preflight without origin header",
			method:     http.MethodOptions,
			reqHeaders: nil,
			wantCode:   http.StatusForbidden,
		},
		{
			name:   "actual POST request with allowed origin",
			method: http.MethodPost,
			reqHeaders: map[string]string{
				"Origin": "http://localhost:3000",
			},
			wantCode: http.StatusOK,
			wantHeaders: map[string]string{
				"Access-Control-Allow-Origin":      "http://localhost:3000",
				"Access-Control-Allow-Credentials": "true",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := CreateServer(allowedOrigins)
			r.GET("/test", func(c *gin.Context) { c.Status(200) })
			r.POST("/test", func(c *gin.Context) { c.Status(200) })

			req := httptest.NewRequest(tc.method, "/test", nil)

			for k, v := range tc.reqHeaders {
				req.Header.Set(k, v)
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tc.wantCode, w.Code)

			for k, v := range tc.wantHeaders {
				assert.Equal(t, v, w.Header().Get(k), "Header %s mismatch", k)
			}
		})
	}
}

func TestWebsocketHandshakeHeaders(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	r := CreateServer([]string{"http://localhost:3000"})

	r.GET("/ws-test", func(c *gin.Context) {
		c.Status(http.StatusSwitchingProtocols)
	})

	tests := []struct {
		name           string
		reqHeaders     map[string]string
		expectedStatus int
	}{
		{
			name: "Valid Websocket Handshake from Allowed Origin",
			reqHeaders: map[string]string{
				"Origin":                "http://localhost:3000",
				"Connection":            "Upgrade",
				"Upgrade":               "websocket",
				"Sec-WebSocket-Version": "13",
				"Sec-WebSocket-Key":     "dGhlIHNhbXBsZSBub25jZQ==",
			},
			expectedStatus: http.StatusSwitchingProtocols,
		},
		{
			name: "Websocket Handshake from Evil Origin",
			reqHeaders: map[string]string{
				"Origin":                "http://evil.com",
				"Connection":            "Upgrade",
				"Upgrade":               "websocket",
				"Sec-WebSocket-Version": "13",
				"Sec-WebSocket-Key":     "dGhlIHNhbXBsZSBub25jZQ==",
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ws-test", nil)
			for k, v := range tc.reqHeaders {
				req.Header.Set(k, v)
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}
