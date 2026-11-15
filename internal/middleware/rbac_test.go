package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type TestClaims struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func createTestTokenForRBAC(userID uint, role string, expiration time.Time, secret string) string {
	claims := &TestClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiration),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

func TestRBAC_AdminCanAccess(t *testing.T) {
	router := gin.New()
	router.Use(AuthMiddleware())
	router.POST("/admin/users", RequireRole("admin"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "admin action"})
	})

	token := createTestTokenForRBAC(1, "admin", time.Now().Add(24*time.Hour), "default-secret-key")
	req := httptest.NewRequest("POST", "/admin/users", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestRBAC_ReaderCannotAccessAdmin(t *testing.T) {
	router := gin.New()
	router.Use(AuthMiddleware())
	router.POST("/admin/users", RequireRole("admin"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "admin action"})
	})

	token := createTestTokenForRBAC(2, "reader", time.Now().Add(24*time.Hour), "default-secret-key")
	req := httptest.NewRequest("POST", "/admin/users", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403, got %d", w.Code)
	}
}

func TestRBAC_LibrarianCanAccessStaffEndpoint(t *testing.T) {
	router := gin.New()
	router.Use(AuthMiddleware())
	router.POST("/staff/books", RequireRole("librarian", "admin"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "staff action"})
	})

	token := createTestTokenForRBAC(1, "librarian", time.Now().Add(24*time.Hour), "default-secret-key")
	req := httptest.NewRequest("POST", "/staff/books", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestRBAC_ReaderCannotAccessStaffEndpoint(t *testing.T) {
	router := gin.New()
	router.Use(AuthMiddleware())
	router.POST("/staff/books", RequireRole("librarian", "admin"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "staff action"})
	})

	token := createTestTokenForRBAC(2, "reader", time.Now().Add(24*time.Hour), "default-secret-key")
	req := httptest.NewRequest("POST", "/staff/books", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403, got %d", w.Code)
	}
}

func TestRBAC_MultipleRolePermissions(t *testing.T) {
	tests := []struct {
		name       string
		role       string
		statusCode int
	}{
		{"Admin access", "admin", http.StatusOK},
		{"Librarian access", "librarian", http.StatusOK},
		{"Reader denied", "reader", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(AuthMiddleware())
			router.POST("/protected", RequireRole("admin", "librarian"), func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			token := createTestTokenForRBAC(1, tt.role, time.Now().Add(24*time.Hour), "default-secret-key")
			req := httptest.NewRequest("POST", "/protected", nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.statusCode {
				t.Errorf("Expected %d, got %d", tt.statusCode, w.Code)
			}
		})
	}
}

func TestRBAC_MissingTokenReturns401(t *testing.T) {
	router := gin.New()
	router.Use(AuthMiddleware())
	router.POST("/protected", RequireRole("admin"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("POST", "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}
}

func TestRBAC_InvalidTokenReturns401(t *testing.T) {
	router := gin.New()
	router.Use(AuthMiddleware())
	router.POST("/protected", RequireRole("admin"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("POST", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}
}

func TestRBAC_ExecutionHaltedOnForbidden(t *testing.T) {
	executed := false
	router := gin.New()
	router.Use(AuthMiddleware())
	router.POST("/protected", RequireRole("admin"), func(c *gin.Context) {
		executed = true
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	token := createTestTokenForRBAC(2, "reader", time.Now().Add(24*time.Hour), "default-secret-key")
	req := httptest.NewRequest("POST", "/protected", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if executed {
		t.Error("Handler should not execute after Forbidden response")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403, got %d", w.Code)
	}
}
