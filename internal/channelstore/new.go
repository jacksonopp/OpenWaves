package channelstore

import "gorm.io/gorm"

// New returns a GORM-backed Store. The caller is responsible for calling
// db.AutoMigrate(&Station{}) before using the store.
func New(db *gorm.DB) Store {
	return &GORMStore{db: db}
}
