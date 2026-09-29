package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fs-erp/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type GoodsHandler struct {
	db *gorm.DB
}

func NewGoodsHandler(db *gorm.DB) *GoodsHandler {
	return &GoodsHandler{db: db}
}

type PageResult struct {
	Total int64         `json:"total"`
	List  []model.Goods `json:"list"`
}

// ListGoods 分页查询商品列表
// 查询条件：goodsNo(货号)、name(名称)、brand(品牌)
func (h *GoodsHandler) ListGoods(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 500 {
		size = 10
	}

	query := h.buildGoodsQuery(c)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "查询失败: " + err.Error()})
		return
	}

	var list []model.Goods
	if err := query.Order("create_time DESC, id DESC").
		Offset((page - 1) * size).Limit(size).Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "查询失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": PageResult{Total: total, List: list}})
}

func (h *GoodsHandler) buildGoodsQuery(c *gin.Context) *gorm.DB {
	query := h.db.Model(&model.Goods{}).Where("deleted = 0")
	if v := strings.TrimSpace(c.Query("goodsNo")); v != "" {
		query = query.Where("goods_no LIKE ?", "%"+v+"%")
	}
	if v := strings.TrimSpace(c.Query("name")); v != "" {
		query = query.Where("name LIKE ?", "%"+v+"%")
	}
	if v := strings.TrimSpace(c.Query("brand")); v != "" {
		query = query.Where("brand = ?", v)
	}
	return query
}

type GoodsReq struct {
	Name       string  `json:"name" binding:"required"`
	GoodsNo    string  `json:"goodsNo" binding:"required"`
	CostPrice  float64 `json:"costPrice"`
	SalesPrice float64 `json:"salesPrice"`
	Stock      int     `json:"stock"`
	Brand      string  `json:"brand" binding:"required"`
	URL        string  `json:"url"`
	Status     *int8   `json:"status"`
	CreateBy   string  `json:"createBy"`
	UpdateBy   string  `json:"updateBy"`
}

// CreateGoods 新增商品（货号+名称+品牌唯一）
func (h *GoodsHandler) CreateGoods(c *gin.Context) {
	var req GoodsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "参数错误: " + err.Error()})
		return
	}
	if exists := h.existsGoods(req.GoodsNo, req.Name, req.Brand, 0); exists {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "该货号+名称+品牌组合已存在"})
		return
	}
	now := model.LocalTime(time.Now())
	g := model.Goods{
		Name: req.Name, GoodsNo: req.GoodsNo,
		CostPrice: req.CostPrice, SalesPrice: req.SalesPrice,
		Stock: req.Stock, Brand: req.Brand, URL: req.URL,
		Status: 0, CreateBy: req.CreateBy, UpdateBy: req.CreateBy,
		CreateTime: now, UpdateTime: now,
	}
	if req.Status != nil {
		g.Status = *req.Status
	}
	if err := h.db.Create(&g).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "保存失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": g})
}

// UpdateGoods 编辑商品
func (h *GoodsHandler) UpdateGoods(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "无效的商品ID"})
		return
	}
	var req GoodsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "参数错误: " + err.Error()})
		return
	}
	var g model.Goods
	if err := h.db.Where("id = ? AND deleted = 0", id).First(&g).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "msg": "商品不存在"})
		return
	}
	if exists := h.existsGoods(req.GoodsNo, req.Name, req.Brand, id); exists {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "该货号+名称+品牌组合已存在"})
		return
	}
	updates := map[string]any{
		"name": req.Name, "goods_no": req.GoodsNo,
		"cost_price": req.CostPrice, "sales_price": req.SalesPrice,
		"stock": req.Stock, "brand": req.Brand, "url": req.URL,
		"update_by": req.UpdateBy, "update_time": time.Now(),
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if err := h.db.Model(&g).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "更新失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0})
}

// UpdateStock 行内修改库存
func (h *GoodsHandler) UpdateStock(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "无效的商品ID"})
		return
	}
	var req struct {
		Stock int `json:"stock"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Stock < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "库存必须为不小于0的整数"})
		return
	}
	result := h.db.Model(&model.Goods{}).Where("id = ? AND deleted = 0", id).
		Updates(map[string]any{"stock": req.Stock, "update_time": time.Now()})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "更新失败: " + result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "msg": "商品不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0})
}

// DeleteGoods 逻辑删除商品
func (h *GoodsHandler) DeleteGoods(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "无效的商品ID"})
		return
	}
	result := h.db.Model(&model.Goods{}).Where("id = ? AND deleted = 0", id).
		Updates(map[string]any{"deleted": 1, "update_time": time.Now()})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "删除失败: " + result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "msg": "商品不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0})
}

// existsGoods 检查 货号+名称+品牌 组合是否已存在（excludeID 用于编辑时排除自身）
func (h *GoodsHandler) existsGoods(goodsNo, name, brand string, excludeID int64) bool {
	var count int64
	query := h.db.Model(&model.Goods{}).Where("deleted = 0 AND goods_no = ? AND name = ? AND brand = ?", goodsNo, name, brand)
	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}
	query.Count(&count)
	return count > 0
}

// templateColumns 导入模板列顺序
var templateColumns = []string{"商品链接", "货号", "成本价", "销售价", "名称", "品牌", "库存"}

// importRequiredColumns 导入必填列
var importRequiredColumns = []string{"货号", "商品链接"}

// defaultImportStock 导入时库存缺省值
const defaultImportStock = 100

// ImportGoods Excel 导入：货号+商品链接不为空即导入；以 货号+名称+品牌+链接
// （名称/品牌为空按空串参与匹配）为唯一键，存在则覆盖，否则新增
func (h *GoodsHandler) ImportGoods(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "请选择要导入的Excel文件"})
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "读取文件失败: " + err.Error()})
		return
	}
	defer f.Close()

	excel, err := excelize.OpenReader(f)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "解析Excel失败: " + err.Error()})
		return
	}
	defer excel.Close()

	sheet := excel.GetSheetList()[0]
	rows, err := excel.GetRows(sheet)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "读取Excel数据失败: " + err.Error()})
		return
	}
	if len(rows) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "Excel中没有可导入的数据"})
		return
	}

	// 解析表头，按列名定位（兼容列顺序调整）
	headerIdx := map[string]int{}
	for i, cell := range rows[0] {
		headerIdx[strings.TrimSpace(cell)] = i
	}
	for _, col := range importRequiredColumns { // 货号、名称、品牌为必填列
		if _, ok := headerIdx[col]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": fmt.Sprintf("Excel缺少必填列：%s", col)})
			return
		}
	}

	now := model.LocalTime(time.Now())

	// 一次性预加载现有商品的唯一键索引，避免逐行 SELECT（性能优化）
	// 唯一键：货号+名称+品牌+链接（名称/品牌为空按空串参与匹配）
	type existKey struct{ goodsNo, name, brand, url string }
	existing := make(map[existKey]model.Goods)
	var allGoods []model.Goods
	if err := h.db.Select("id", "goods_no", "name", "brand", "url").
		Where("deleted = 0").Find(&allGoods).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "读取现有商品失败: " + err.Error()})
		return
	}
	for _, g := range allGoods {
		existing[existKey{g.GoodsNo, g.Name, g.Brand, g.URL}] = model.Goods{ID: g.ID, GoodsNo: g.GoodsNo}
	}

	toInsert := make([]model.Goods, 0, len(rows))
	toUpdate := make([]model.Goods, 0, len(rows))
	pendingIdx := make(map[existKey]int) // 同一文件内重复行定位到待插入记录
	newBrands := map[string]struct{}{}
	var errs []string
	var duplicates, blankRows int
	for i, row := range rows[1:] {
		lineNo := i + 2 // Excel 中的实际行号（含表头）
		get := func(col string) string {
			idx, ok := headerIdx[col]
			if !ok || idx >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[idx])
		}
		// 空行直接忽略
		if allCellsEmpty(row) {
			blankRows++
			continue
		}
		goodsNo, url := get("货号"), get("商品链接")
		if goodsNo == "" || url == "" {
			errs = append(errs, fmt.Sprintf("第%d行：货号和商品链接不能为空", lineNo))
			continue
		}
		name, brand := get("名称"), get("品牌")
		cost, _ := strconv.ParseFloat(get("成本价"), 64)
		sales, _ := strconv.ParseFloat(get("销售价"), 64)
		stock := defaultImportStock // 库存可为空，默认100
		if v := get("库存"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				stock = n
			}
		}
		if brand != "" {
			newBrands[brand] = struct{}{}
		}

		k := existKey{goodsNo, name, brand, url}
		if idx, ok := pendingIdx[k]; ok {
			// 同文件内重复行：后一行覆盖前一行
			duplicates++
			toInsert[idx].CostPrice, toInsert[idx].SalesPrice = cost, sales
			toInsert[idx].Stock = stock
			continue
		}
		if exist, ok := existing[k]; ok {
			// 已存在则覆盖
			exist.CostPrice, exist.SalesPrice, exist.Stock = cost, sales, stock
			exist.UpdateBy, exist.UpdateTime = "导入", now
			toUpdate = append(toUpdate, exist)
		} else {
			// 不存在则新增
			pendingIdx[k] = len(toInsert)
			toInsert = append(toInsert, model.Goods{
				Name: name, GoodsNo: goodsNo, Brand: brand,
				CostPrice: cost, SalesPrice: sales, Stock: stock, URL: url,
				CreateBy: "导入", UpdateBy: "导入", CreateTime: now, UpdateTime: now,
			})
		}
	}

	// 单事务批量写入，提交一次落盘（性能优化）
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()
	if len(toInsert) > 0 {
		if err := tx.CreateInBatches(toInsert, 500).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "批量新增失败: " + err.Error()})
			return
		}
	}
	for _, g := range toUpdate {
		if err := tx.Model(&model.Goods{}).Where("id = ?", g.ID).Updates(map[string]any{
			"cost_price": g.CostPrice, "sales_price": g.SalesPrice, "stock": g.Stock,
			"url": g.URL, "update_by": g.UpdateBy, "update_time": time.Time(g.UpdateTime),
		}).Error; err != nil {
			errs = append(errs, fmt.Sprintf("货号[%s]更新失败: %v", g.GoodsNo, err))
		}
	}
	// 新品牌写入品牌枚举表
	for name := range newBrands {
		var count int64
		tx.Model(&model.Brand{}).Where("name = ?", name).Count(&count)
		if count == 0 {
			if err := tx.Create(&model.Brand{Name: name, CreateTime: now}).Error; err != nil {
				errs = append(errs, fmt.Sprintf("品牌[%s]写入失败：%v", name, err))
			}
		}
	}
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "导入提交失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"added": len(toInsert), "updated": len(toUpdate), "errors": errs,
		"duplicates": duplicates, "blankRows": blankRows, "totalRows": len(rows) - 1,
	}})
}

// ExportGoods 导出商品列表（跟随当前查询条件）
func (h *GoodsHandler) ExportGoods(c *gin.Context) {
	var list []model.Goods
	if err := h.buildGoodsQuery(c).Order("create_time DESC, id DESC").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "查询失败: " + err.Error()})
		return
	}

	excel := excelize.NewFile()
	defer excel.Close()
	sheet := excel.GetSheetList()[0]
	// 列顺序与导入模板一致，导出文件可直接再导入
	headers := templateColumns
	for i, title := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		excel.SetCellValue(sheet, cell, title)
	}
	for i, g := range list {
		row := i + 2
		set := func(col int, val any) {
			cell, _ := excelize.CoordinatesToCellName(col, row)
			excel.SetCellValue(sheet, cell, val)
		}
		set(1, g.URL)
		set(2, g.GoodsNo)
		set(3, g.CostPrice)
		set(4, g.SalesPrice)
		set(5, g.Name)
		set(6, g.Brand)
		set(7, g.Stock)
	}

	c.Header("Content-Disposition", `attachment; filename="goods_`+time.Now().Format("20060102150405")+`.xlsx"`)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	if err := excel.Write(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "导出失败: " + err.Error()})
	}
}

// allCellsEmpty 判断一行是否所有单元格都为空
func allCellsEmpty(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

// ImportTemplate 下载导入模板
func (h *GoodsHandler) ImportTemplate(c *gin.Context) {
	excel := excelize.NewFile()
	defer excel.Close()
	sheet := excel.GetSheetList()[0]
	for i, title := range templateColumns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		excel.SetCellValue(sheet, cell, title)
	}
	// 示例行（顺序与模板一致：链接、货号、成本价、销售价、名称、品牌、库存）
	example := []any{"https://example.com/p/1", "SP001", 4999.00, 5999.00, "iPhone 15", "Apple", 100}
	for i, v := range example {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		excel.SetCellValue(sheet, cell, v)
	}

	c.Header("Content-Disposition", `attachment; filename="goods_import_template.xlsx"`)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	if err := excel.Write(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "下载失败: " + err.Error()})
	}
}
