package channelstore

import "gorm.io/gorm"

type GORMStore struct{ db *gorm.DB }

func (g *GORMStore) Create(s Station) error {
	return g.db.Create(&s).Error
}

func (g *GORMStore) Delete(username string) error {
	return g.db.Where("username = ?", username).Delete(&Station{}).Error
}

func (g *GORMStore) List() ([]Station, error) {
	var stations []Station
	if err := g.db.Find(&stations).Error; err != nil {
		return nil, err
	}
	return stations, nil
}
