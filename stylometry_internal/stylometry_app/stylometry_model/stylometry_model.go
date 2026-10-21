package model

import "time"

// Status describes the lifecycle state of a sample.
type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusDeleted   Status = "deleted"
)

// User is an application user.
type User struct {
	ID       uint   `gorm:"primaryKey;column:ID"`
	Name     string `gorm:"column:Name;size:120;not null"`
	Password string `gorm:"column:Password;size:255;not null"`
}

// TableName returns the exact PostgreSQL table name for users.
func (User) TableName() string {
	return "Users"
}

// Sample is a text sample used for stylometric analysis.
type Sample struct {
	ID                    uint       `gorm:"primaryKey;column:ID"`
	TextCreator           string     `gorm:"column:TextCreator;size:160;not null"`
	ShortDescription      string     `gorm:"column:ShortDescription;size:800"`
	Status                Status     `gorm:"column:Status;type:varchar(20);not null;default:'draft';index"`
	ImageURL              string     `gorm:"column:ImageURL;size:500;not null;default:'/stylometry_static/stylometry_default_image.svg'"`
	VideoURL              string     `gorm:"column:VideoURL;size:500;not null;default:'/stylometry_static/stylometry_default_video.svg'"`
	AverageSentenceLength float64    `gorm:"column:AverageSentenceLength"`
	ConjunctionFrequency  float64    `gorm:"column:ConjunctionFrequency"`
	CreatedAt             *time.Time `gorm:"column:CreatedAt;autoCreateTime"`
	UpdatedAt             *time.Time `gorm:"column:UpdatedAt"`
	CreatorID             uint       `gorm:"column:CreatorID;not null;index"`
	Creator               User       `gorm:"foreignKey:CreatorID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
	Likes                 []Like     `gorm:"foreignKey:SampleID"`
}

// TableName returns the exact PostgreSQL table name for text samples.
func (Sample) TableName() string {
	return "StylometrySamples"
}

// LikesCount returns the number of likes attached to a sample.
func (sample Sample) LikesCount() int {
	return len(sample.Likes)
}

// IsDraft reports whether a sample is a draft.
func (sample Sample) IsDraft() bool {
	return sample.Status == StatusDraft
}

// Like is the many-to-many link between a user and a sample.
type Like struct {
	ID       uint   `gorm:"primaryKey;column:ID"`
	UserID   uint   `gorm:"column:UserID;not null"`
	SampleID uint   `gorm:"column:SampleID;not null"`
	User     User   `gorm:"foreignKey:UserID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
	Sample   Sample `gorm:"foreignKey:SampleID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
}

// TableName returns the exact PostgreSQL table name for likes.
func (Like) TableName() string {
	return "Likes"
}
