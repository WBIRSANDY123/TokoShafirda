package models

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Product struct {
	ID               string `gorm:"size:36;not null;uniqueIndex;primary_key"`
	ParentID         string `gorm:"size:36;index"`
	Name             string `gorm:"size:255"`
	SATUAN1          string `gorm:"size:255"`
	SATUAN2          string `gorm:"size:255"`
	SATUAN3          string `gorm:"size:255"`
	KONVERSI1        int
	KONVERSI2        int
	KONVERSI3        int
	HARGAPOKOK1      decimal.Decimal `gorm:"type:decimal(16,2);"`
	HARGAPOKOK2      decimal.Decimal `gorm:"type:decimal(16,2);"`
	HARGAPOKOK3      decimal.Decimal `gorm:"type:decimal(16,2);"`
	HJ1              decimal.Decimal `gorm:"type:decimal(16,2);"`
	HJ2              decimal.Decimal `gorm:"type:decimal(16,2);"`
	HJ3              decimal.Decimal `gorm:"type:decimal(16,2);"`
	HJ2_1            decimal.Decimal `gorm:"type:decimal(16,2);"`
	HJ2_2            decimal.Decimal `gorm:"type:decimal(16,2);"`
	HJ2_3            decimal.Decimal `gorm:"type:decimal(16,2);"`
	Stock            int
	Supplier         string `gorm:"type:text"`
	ProductImages    []ProductImage
	Categories       string          `gorm:"size:255"`
	Sku              string          `gorm:"size:100;index"`
	Slug             string          `gorm:"size:255"`
	Price            decimal.Decimal `gorm:"type:decimal(16,2);"`
	Weight           decimal.Decimal `gorm:"type:decimal(10,2);"`
	ShortDescription string          `gorm:"type:text"`
	Description      string          `gorm:"type:text"`
	Status           int             `gorm:"default:0"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt
}

func (p *Product) GetProducts(db *gorm.DB, perPage int, page int) (*[]Product, int64, error) {
	var err error
	var products []Product
	var count int64

	err = db.Debug().Model(&Product{}).Count(&count).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage

	err = db.Debug().Preload("ProductImages").Model(&Product{}).Order("stock desc").Limit(perPage).Offset(offset).Find(&products).Error
	if err != nil {
		return nil, 0, err
	}

	return &products, count, nil
}

func (p *Product) FindBySlug(db *gorm.DB, slug string) (*Product, error) {
	var err error
	var product Product

	err = db.Debug().Preload("ProductImages").Model(&Product{}).Where("slug = ?", slug).First(&product).Error
	if err != nil {
		return nil, err
	}

	return &product, nil
}

func (p *Product) FindByID(db *gorm.DB, productID string) (*Product, error) {
	var err error
	var product Product

	err = db.Debug().Preload("ProductImages").Model(&Product{}).Where("id = ?", productID).First(&product).Error
	if err != nil {
		return nil, err
	}

	return &product, nil
}

func (p *Product) SearchProducts(db *gorm.DB, query string, perPage int, page int) (*[]Product, int64, error) {
	var err error
	var products []Product
	var count int64

	searchQuery := "%" + query + "%"

	queryBuilder := db.Debug().Model(&Product{}).Where(
		"LOWER(name) LIKE LOWER(?) OR LOWER(categories) LIKE LOWER(?) OR LOWER(satuan1) LIKE LOWER(?) OR LOWER(satuan2) LIKE LOWER(?) OR LOWER(satuan3) LIKE LOWER(?)",
		searchQuery, searchQuery, searchQuery, searchQuery, searchQuery,
	)

	err = queryBuilder.Count(&count).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage

	err = queryBuilder.Preload("ProductImages").Order("stock desc").Limit(perPage).Offset(offset).Find(&products).Error
	if err != nil {
		return nil, 0, err
	}

	return &products, count, nil
}
