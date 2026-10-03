package model

import "time"

// StylometrySampleStatus — отдельный строковый тип для статуса карточки.
type StylometrySampleStatus string

const (
	StatusDraft     StylometrySampleStatus = "draft"
	StatusPublished StylometrySampleStatus = "published"
	StatusDeleted   StylometrySampleStatus = "deleted"
)

// StylometrySampleUser — модель пользователя приложения.
type StylometrySampleUser struct {
	StylometrySampleID        uint      `gorm:"primaryKey;column:id"`
	StylometrySampleName      string    `gorm:"column:name;size:120;not null"`
	StylometrySampleEmail     string    `gorm:"column:email;size:180;not null;uniqueIndex"`
	StylometrySampleCreatedAt time.Time `gorm:"column:created_at;not null"`
}

// TableName задаёт точное имя таблицы пользователей.
func (StylometrySampleUser) TableName() string {
	return "stylometry_users"
}

// StylometrySampleTextSample — главная модель лабораторной: карточка эталонного текста.
type StylometrySampleTextSample struct {
	StylometrySampleID         uint                   `gorm:"primaryKey;column:id"`
	StylometrySampleTitle      string                 `gorm:"column:title;size:160;not null"`
	StylometrySampleDescr      string                 `gorm:"column:short_description;size:800"`
	StylometrySampleStatus     StylometrySampleStatus `gorm:"column:status;type:varchar(20);not null;default:'draft';index"`
	StylometrySampleImageURL   string                 `gorm:"column:image_url;size:500"`
	StylometrySampleVideoURL   string                 `gorm:"column:video_url;size:500"`
	StylometrySampleAuthorName string                 `gorm:"column:author_name;size:160;not null"`
	StylometrySampleAvgSentLen float64                `gorm:"column:avg_sentence_length;not null;default:0"`
	StylometrySampleConjFreq   float64                `gorm:"column:conjunction_frequency;not null;default:0"`
	StylometrySampleCreatedAt  time.Time              `gorm:"column:created_at;not null"`
	StylometrySampleFormedAt   *time.Time             `gorm:"column:formed_at"`
	StylometrySampleCreatorID  uint                   `gorm:"column:creator_id;not null;index;uniqueIndex:idx_one_draft_per_user,where:status = 'draft'"`
	StylometrySampleCreator    StylometrySampleUser   `gorm:"foreignKey:StylometrySampleCreatorID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
	StylometrySampleLikes      []StylometrySampleLike `gorm:"foreignKey:StylometrySampleTextSampleID"`
}

// TableName задаёт точное имя таблицы карточек.
func (StylometrySampleTextSample) TableName() string {
	return "stylometry_text_samples"
}

// LikesCount возвращает количество лайков у карточки.
func (stylometry_sample StylometrySampleTextSample) LikesCount() int {
	return len(stylometry_sample.StylometrySampleLikes)
}

// IsDraft сообщает, является ли карточка черновиком.
func (stylometry_sample StylometrySampleTextSample) IsDraft() bool {
	return stylometry_sample.StylometrySampleStatus == StatusDraft
}

// StylometrySampleLike — модель лайка, связь многие-ко-многим пользователь-карточка.
type StylometrySampleLike struct {
	StylometrySampleLikeID       uint                       `gorm:"primaryKey;column:id"`
	StylometrySampleUserID       uint                       `gorm:"column:user_id;not null;uniqueIndex:idx_user_sample_like"`
	StylometrySampleTextSampleID uint                       `gorm:"column:sample_id;not null;uniqueIndex:idx_user_sample_like"`
	StylometrySampleUser         StylometrySampleUser       `gorm:"foreignKey:StylometrySampleUserID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
	StylometrySampleTextSample   StylometrySampleTextSample `gorm:"foreignKey:StylometrySampleTextSampleID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
	StylometrySampleCreatedAt    time.Time                  `gorm:"column:created_at;not null"`
}

// TableName задаёт точное имя таблицы лайков.
func (StylometrySampleLike) TableName() string {
	return "stylometry_likes"
}
