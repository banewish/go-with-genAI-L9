package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"homework/internal/model"
)

type MockUserRepository struct {
	users  map[uint]*model.User
	nextID uint
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users:  make(map[uint]*model.User),
		nextID: 1,
	}
}

func (m *MockUserRepository) Create(user *model.User) error {
	for _, u := range m.users {
		if u.Email == user.Email {
			return gorm.ErrDuplicatedKey
		}
	}
	user.ID = m.nextID
	m.nextID++
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) GetByEmail(email string) (*model.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *MockUserRepository) GetByID(id uint) (*model.User, error) {
	if user, ok := m.users[id]; ok {
		return user, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *MockUserRepository) Update(user *model.User) error {
	if _, ok := m.users[user.ID]; ok {
		m.users[user.ID] = user
		return nil
	}
	return gorm.ErrRecordNotFound
}

func (m *MockUserRepository) List() ([]model.User, error) {
	var users []model.User
	for _, u := range m.users {
		users = append(users, *u)
	}
	return users, nil
}

func (m *MockUserRepository) UpdateRole(id uint, role string) error {
	if user, ok := m.users[id]; ok {
		user.Role = role
		return nil
	}
	return gorm.ErrRecordNotFound
}

func TestAuthHandler_Register_Success(t *testing.T) {
	repo := NewMockUserRepository()
	handler := NewAuthHandler(repo)
	router := gin.New()
	router.POST("/api/v1/auth/register", handler.Register)

	reqBody := model.RegisterRequest{
		Email:    "user@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["user"] == nil {
		t.Error("Expected user in response")
	}
}

func TestAuthHandler_Register_DuplicateEmail(t *testing.T) {
	repo := NewMockUserRepository()
	handler := NewAuthHandler(repo)

	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	repo.Create(&model.User{
		Email:        "user@example.com",
		PasswordHash: string(hash),
		Role:         "reader",
	})

	router := gin.New()
	router.POST("/api/v1/auth/register", handler.Register)

	reqBody := model.RegisterRequest{
		Email:    "user@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
	}
}

func TestAuthHandler_Login_Success(t *testing.T) {
	repo := NewMockUserRepository()
	handler := NewAuthHandler(repo)

	password := "password123"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	user := &model.User{
		Email:        "user@example.com",
		PasswordHash: string(hash),
		Role:         "reader",
	}
	repo.Create(user)

	router := gin.New()
	router.POST("/api/v1/auth/login", handler.Login)

	reqBody := model.LoginRequest{
		Email:    "user@example.com",
		Password: password,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp model.LoginResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Token == "" {
		t.Error("Expected token in response")
	}
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
	}{
		{"Unknown email", "nonexistent@example.com", "password123"},
		{"Wrong password", "user@example.com", "wrongpassword"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockUserRepository()
			handler := NewAuthHandler(repo)

			password := "password123"
			hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			user := &model.User{
				Email:        "user@example.com",
				PasswordHash: string(hash),
				Role:         "reader",
			}
			repo.Create(user)

			router := gin.New()
			router.POST("/api/v1/auth/login", handler.Login)

			reqBody := model.LoginRequest{
				Email:    tt.email,
				Password: tt.password,
			}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("Expected status 401, got %d for %s", w.Code, tt.name)
			}

			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)
			if resp["error"] != "invalid credentials" {
				t.Errorf("Expected 'invalid credentials' error, got %v", resp["error"])
			}
		})
	}
}

func TestAuthHandler_PasswordHashNotExposed(t *testing.T) {
	repo := NewMockUserRepository()
	handler := NewAuthHandler(repo)

	password := "password123"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	user := &model.User{
		Email:        "user@example.com",
		PasswordHash: string(hash),
		Role:         "reader",
	}
	repo.Create(user)

	router := gin.New()
	router.POST("/api/v1/auth/login", handler.Login)

	reqBody := model.LoginRequest{
		Email:    "user@example.com",
		Password: password,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Check that response doesn't contain password hash
	if bytes.Contains(w.Body.Bytes(), []byte(hash)) {
		t.Error("Password hash exposed in response")
	}
}

func createTestToken(userID uint, role string, expiration time.Time) string {
	claims := &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiration),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte("default-secret-key"))
	return tokenString
}
