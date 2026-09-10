package database

import (
	"fmt"

	"wiki/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(conf *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable", conf.DatabaseHost, conf.DatabaseUser, conf.DatabasePassword, conf.DatabaseName, conf.DatabasePort)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	err = sqlDB.Ping()
	if err != nil {
		return nil, err
	}
	return db, err
}
