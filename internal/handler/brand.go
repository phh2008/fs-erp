package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/fs-erp/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type BrandHandler struct {
	db *gorm.DB
}

func NewBrandHandler(db *gorm.DB) *BrandHandler {
	return &BrandHandler{db: db}
}

// ListBrands 品牌下拉枚举列表
func (h *BrandHandler) ListBrands(c *gin.Context) {
	var brands []model.Brand
	if err := h.db.Order("id ASC").Find(&brands).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "查询失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": brands})
}

// CreateBrand 新增品牌
func (h *BrandHandler) CreateBrand(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "请输入品牌名称"})
		return
	}
	name := strings.TrimSpace(req.Name)
	var count int64
	h.db.Model(&model.Brand{}).Where("name = ?", name).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "品牌已存在"})
		return
	}
	b := model.Brand{Name: name, CreateTime: model.LocalTime(time.Now())}
	if err := h.db.Create(&b).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "保存失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": b})
}
