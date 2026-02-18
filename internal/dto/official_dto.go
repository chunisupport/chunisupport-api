package dto

import "time"

// OfficialSongWithGenreDTO はジャンルIDを含む公式楽曲データの転送オブジェクトです
type OfficialSongWithGenreDTO struct {
	ID         int       `db:"id"`
	OfficialID string    `db:"official_id"`
	Catname    string    `db:"catname"`
	GenreID    int       `db:"genre_id"`
	Newflag    string    `db:"newflag"`
	Title      string    `db:"title"`
	Reading    *string   `db:"reading"`
	Artist     string    `db:"artist"`
	LevBas     *string   `db:"lev_bas"`
	LevAdv     *string   `db:"lev_adv"`
	LevExp     *string   `db:"lev_exp"`
	LevMas     *string   `db:"lev_mas"`
	LevUlt     *string   `db:"lev_ult"`
	Attribute  *string   `db:"attribute"`
	LevelStar  *string   `db:"level_star"`
	Image      string    `db:"image"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}
