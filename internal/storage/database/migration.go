package database

import (
	"gorm.io/gorm"
	"internship-effective-mobile/internal/models"
	"log"
)

func Migrate(db *gorm.DB) {

	if err := db.AutoMigrate(&models.Song{}); err != nil {
		log.Fatal("migrate failed:  ", err)
	}
}
