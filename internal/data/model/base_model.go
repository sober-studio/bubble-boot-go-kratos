package model

import (
	"time"

	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/idgen"
	"gorm.io/gorm"
)

type BaseModel struct {
	ID        int64          `gorm:"column:id;primaryKey"`
	CreatedAt time.Time      `gorm:"column:created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

var globalIDGen idgen.IDGenerator

func SetIDGenerator(g idgen.IDGenerator) { globalIDGen = g }

func NextID() (int64, error) {
	if globalIDGen == nil {
		return 0, nil
	}
	return globalIDGen.NextID()
}

func (m *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if m.ID != 0 {
		return nil
	}
	if globalIDGen == nil {
		return nil
	}
	id, err := globalIDGen.NextID()
	if err != nil {
		return err
	}
	m.ID = id
	return nil
}
