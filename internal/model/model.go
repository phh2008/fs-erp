package model

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// LocalTime JSON 序列化为 "2006-01-02 15:04:05"
type LocalTime time.Time

func (t LocalTime) MarshalJSON() ([]byte, error) {
	stamp := fmt.Sprintf("\"%s\"", time.Time(t).Format("2006-01-02 15:04:05"))
	return []byte(stamp), nil
}

func (t *LocalTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*t = LocalTime(time.Time{})
		return nil
	}
	tt, err := time.ParseInLocation(`"2006-01-02 15:04:05"`, string(data), time.Local)
	if err != nil {
		return err
	}
	*t = LocalTime(tt)
	return nil
}

func (t LocalTime) Value() (driver.Value, error) {
	return time.Time(t), nil
}

func (t *LocalTime) Scan(v any) error {
	if v == nil {
		*t = LocalTime(time.Time{})
		return nil
	}
	switch val := v.(type) {
	case time.Time:
		*t = LocalTime(val)
	case []byte:
		tt, err := time.ParseInLocation("2006-01-02 15:04:05", string(val), time.Local)
		if err != nil {
			return err
		}
		*t = LocalTime(tt)
	}
	return nil
}

// Goods 商品
type Goods struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name       string    `gorm:"type:varchar(255);comment:名称" json:"name"`
	GoodsNo    string    `gorm:"type:varchar(50);comment:货号" json:"goodsNo"`
	CostPrice  float64   `gorm:"type:decimal(10,2);comment:成本价" json:"costPrice"`
	SalesPrice float64   `gorm:"type:decimal(10,2);comment:销售价" json:"salesPrice"`
	Stock      int       `gorm:"type:int;comment:库存" json:"stock"`
	Brand      string    `gorm:"type:varchar(50);comment:品牌" json:"brand"`
	URL        string    `gorm:"type:varchar(1000);comment:商品链接" json:"url"`
	Status     int8      `gorm:"type:tinyint(1);default:0;comment:状态：0-已上架，1-已下架" json:"status"`
	CreateBy   string    `gorm:"type:varchar(255);comment:创建人" json:"createBy"`
	UpdateBy   string    `gorm:"type:varchar(255);comment:更新人" json:"updateBy"`
	CreateTime LocalTime `gorm:"type:datetime;comment:创建日期" json:"createTime"`
	UpdateTime LocalTime `gorm:"type:datetime;comment:更新日期" json:"updateTime"`
	Deleted    int8      `gorm:"type:tinyint(1);default:0;comment:删除：0-否，1-是" json:"-"`
}

func (Goods) TableName() string { return "gds_goods" }

// IsOnShelf 是否上架
func (g Goods) IsOnShelf() bool { return g.Status == 0 }

// Brand 品牌枚举表
type Brand struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name       string    `gorm:"type:varchar(50);uniqueIndex;comment:品牌名称" json:"name"`
	CreateTime LocalTime `gorm:"type:datetime;comment:创建日期" json:"createTime"`
}

func (Brand) TableName() string { return "sys_brand" }
