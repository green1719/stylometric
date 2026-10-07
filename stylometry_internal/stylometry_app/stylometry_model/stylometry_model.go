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
	StylometrySampleID       uint   `gorm:"primaryKey;column:StylometrySampleID"`
	StylometrySampleName     string `gorm:"column:StylometrySampleName;size:120;not null"`
	StylometrySamplePassword string `gorm:"column:StylometrySamplePassword;size:255;not null;default:'stylometry123'"`
}

// TableName задаёт точное имя таблицы пользователей.
func (StylometrySampleUser) TableName() string {
	return "StylometrySampleUsers"
}

// StylometrySampleTextSample: карточка эталонного текста.
type StylometrySampleTextSample struct {
	StylometrySampleID                    uint                   `gorm:"primaryKey;column:StylometrySampleID"`
	StylometrySampleTextCreator           string                 `gorm:"column:StylometrySampleTextCreator;size:160;not null"`
	StylometrySampleDescr                 string                 `gorm:"column:StylometrySampleShortDescription;size:800"`
	StylometrySampleStatus                StylometrySampleStatus `gorm:"column:StylometrySampleStatus;type:varchar(20);not null;default:'draft';index"`
	StylometrySampleImageURL              string                 `gorm:"column:StylometrySampleImageURL;size:500;not null;default:'/stylometry_static/stylometry_default_image.svg'"`
	StylometrySampleVideoURL              string                 `gorm:"column:StylometrySampleVideoURL;size:500;not null;default:'/stylometry_static/stylometry_default_video.svg'"`
	StylometrySampleAverageSentenceLength float64                `gorm:"column:StylometrySampleAverageSentenceLength"`
	StylometrySampleConjunctionFrequency  float64                `gorm:"column:StylometrySampleConjunctionFrequency"`
	StylometrySampleCreatedAt             *time.Time             `gorm:"column:StylometrySampleCreatedAt;autoCreateTime"`
	StylometrySampleUpdatedAt             *time.Time             `gorm:"column:StylometrySampleUpdatedAt"`
	StylometrySampleCreatorID             uint                   `gorm:"column:StylometrySampleCreatorID;not null;index"`
	StylometrySampleCreator               StylometrySampleUser   `gorm:"foreignKey:StylometrySampleCreatorID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
	StylometrySampleLikes                 []StylometrySampleLike `gorm:"foreignKey:StylometrySampleTextSampleID"`
}

// TableName задаёт точное имя таблицы карточек.
func (StylometrySampleTextSample) TableName() string {
	return "StylometrySampleTextSamples"
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
	StylometrySampleLikeID       uint                       `gorm:"primaryKey;column:StylometrySampleLikeID"`
	StylometrySampleUserID       uint                       `gorm:"column:StylometrySampleUserID;not null"`
	StylometrySampleTextSampleID uint                       `gorm:"column:StylometrySampleTextSampleID;not null"`
	StylometrySampleUser         StylometrySampleUser       `gorm:"foreignKey:StylometrySampleUserID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
	StylometrySampleTextSample   StylometrySampleTextSample `gorm:"foreignKey:StylometrySampleTextSampleID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
}

// TableName задаёт точное имя таблицы лайков.
func (StylometrySampleLike) TableName() string {
	return "StylometrySampleLikes"
}
