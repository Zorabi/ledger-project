package test

//
//import (
//	"errors"
//	"github.com/gin-gonic/gin"
//	"github.com/stretchr/testify/assert"
//	"ledger-backend/internal/models"
//	"ledger-backend/internal/services"
//	"net/http"
//	"net/http/httptest"
//	"testing"
//)
//
//type IGiftBookInterface interface {
//	FindById(id uint) (*models.GiftBook, error)
//}
//
//type MockGiftRepo struct{}
//
//func (r *MockGiftRepo) FindById(id uint) (*models.GiftBook, error) {
//	if id == 1 {
//		return &models.GiftBook{Id: 1, UserName: "test"}, nil
//	}
//	return nil, errors.New("not found")
//}
//
//func TestGiftBookService_GetGiftBook(t *testing.T) {
//	mockRepo := &MockGiftRepo{}
//	giftBookService := services.NewGiftBookService(mockRepo)
//
//	user, err := giftBookService.GetUserByID(1)
//	assert.NoError(t, err)
//	assert.Equal(t, "test", user.Username)
//}
//
//
//func TestUserHandler_GetUser(t *testing.T) {
//	// 同样，可以注入 mock 的 service
//	handler := setupMockUserHandler()
//
//	r := gin.Default()
//	r.GET("/users/:id", handler.GetUser)
//
//	// 创建一个模拟的 HTTP 请求
//	req, _ := http.NewRequest("GET", "/users/1", nil)
//
//	// 创建一个响应记录器
//	w := httptest.NewRecorder()
//
//	// 让 Gin Engine 处理这个请求
//	r.ServeHTTP(w, req)
//
//	// 断言结果
//	assert.Equal(t, http.StatusOK, w.Code)
//	assert.Contains(t, w.Body.String(), "test_user")
//}