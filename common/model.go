package common

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"

	"github.com/swiftbit/know-agent/common/utils"
)

type Model struct {
	ID int64 `gorm:"column:id;primaryKey"`
	AuditModel
}

type AuditModel struct {
	CreateTime time.Time             `gorm:"column:create_time;type:datetime;autoCreateTime"`
	UpdateTime time.Time             `gorm:"column:edit_time;type:datetime;autoUpdateTime"`
	Deleted    soft_delete.DeletedAt `gorm:"column:deleted;softDelete:flag"`
}

func (m *Model) BeforeCreate(tx *gorm.DB) error {
	if m.ID == 0 {
		m.ID = utils.GetSnowflakeNextID()
	}
	m.Deleted = 1
	return nil
}

func (m *AuditModel) BeforeCreate(tx *gorm.DB) error {
	m.Deleted = 1
	return nil
}
