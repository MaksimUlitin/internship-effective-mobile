package models

import "time"

type Group struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Name string `gorm:"uniqueIndex" json:"name"`
}

type Song struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	GroupID     uint      `gorm:"index" json:"group_id"`
	Group       Group     `gorm:"foreignKey:GroupID" json:"group"`
	Title       string    `gorm:"index" json:"title"`
	ReleaseDate time.Time `gorm:"type:date;index" json:"release_date"`
	Text        string    `json:"text"`
	Link        string    `gorm:"index" json:"link"`
	CreatedAt   time.Time `json:"created_at"`
}

type SongDetail struct {
	ReleaseDate string `json:"releaseDate"`
	Text        string `json:"text"`
	Link        string `json:"link"`
}

type AddNewSong struct {
	Group string `json:"group"`
	Song  string `json:"song"`
}

type SongUpdateRequest struct {
	GroupID     *int    `json:"group_id"`
	Title       *string `json:"title"`
	ReleaseDate *string `json:"release_date"`
	Lyrics      *string `json:"lyrics"`
	Link        *string `json:"link"`
}
