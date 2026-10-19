package serializer

import (
	"time"

	"stylometry/stylometry_internal/stylometry_app/stylometry_model"
)

type SampleResponseDTO struct {
	ID                    uint       `json:"stylometry_sample_id"`
	TextCreator           string     `json:"stylometry_sample_text_creator"`
	ShortDescription      string     `json:"stylometry_sample_descr"`
	ImageURL              string     `json:"stylometry_sample_image_url"`
	VideoURL              string     `json:"stylometry_sample_video_url"`
	AverageSentenceLength float64    `json:"stylometry_sample_average_sentence_length"`
	ConjunctionFrequency  float64    `json:"stylometry_sample_conjunction_frequency"`
	LikesCount            int        `json:"stylometry_sample_likes_count"`
	IsCurrentUserCreator  int        `json:"stylometry_sample_is_current_user_creator"`
	IsLiked               int        `json:"is_liked"`
	CreatedAt             *time.Time `json:"stylometry_sample_created_at,omitempty"`
	UpdatedAt             *time.Time `json:"stylometry_sample_updated_at,omitempty"`
}

func SampleResponse(stylometry_sample model.Sample, stylometry_current_user_id uint) SampleResponseDTO {
	stylometry_is_creator := 0
	if stylometry_sample.CreatorID == stylometry_current_user_id {
		stylometry_is_creator = 1
	}
	stylometry_is_liked := 0
	for _, stylometry_like := range stylometry_sample.Likes {
		if stylometry_like.UserID == stylometry_current_user_id {
			stylometry_is_liked = 1
			break
		}
	}

	return SampleResponseDTO{
		ID:                    stylometry_sample.ID,
		TextCreator:           stylometry_sample.TextCreator,
		ShortDescription:      stylometry_sample.ShortDescription,
		ImageURL:              stylometry_sample.ImageURL,
		VideoURL:              stylometry_sample.VideoURL,
		AverageSentenceLength: stylometry_sample.AverageSentenceLength,
		ConjunctionFrequency:  stylometry_sample.ConjunctionFrequency,
		LikesCount:            len(stylometry_sample.Likes),
		IsCurrentUserCreator:  stylometry_is_creator,
		IsLiked:               stylometry_is_liked,
		CreatedAt:             stylometry_sample.CreatedAt,
		UpdatedAt:             stylometry_sample.UpdatedAt,
	}
}

func SampleListResponse(stylometry_samples []model.Sample, stylometry_current_user_id uint) []SampleResponseDTO {
	stylometry_response := make([]SampleResponseDTO, 0, len(stylometry_samples))

	for _, stylometry_sample := range stylometry_samples {
		stylometry_response = append(stylometry_response, SampleResponse(stylometry_sample, stylometry_current_user_id))
	}

	return stylometry_response
}

type UserResponseDTO struct {
	ID   uint   `json:"stylometry_sample_id"`
	Name string `json:"stylometry_sample_name"`
}

func UserResponse(stylometry_user model.User) UserResponseDTO {
	return UserResponseDTO{
		ID:   stylometry_user.ID,
		Name: stylometry_user.Name,
	}
}
