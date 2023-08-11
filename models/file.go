package models

import "gorm.io/gorm"

type File struct {
	gorm.Model
	ID          string `gorm:"type:uuid;primary_key;default:uuid_generate_v4();unique"`
	Url         string `gorm:"size:2000;not null"`
	IsConfirmed bool   `gorm:"not null;default:false"`
}
