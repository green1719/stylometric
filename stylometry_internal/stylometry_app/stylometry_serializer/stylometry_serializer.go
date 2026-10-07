package serializer

import (
	"time"

	"stylometry/stylometry_internal/stylometry_app/stylometry_model"
)

type StylometrySampleResponse struct {
	StylometrySampleID                    uint       `json:"stylometry_sample_id"`
	StylometrySampleTextCreator           string     `json:"stylometry_sample_text_creator"`
	StylometrySampleDescr                 string     `json:"stylometry_sample_descr"`
	StylometrySampleImageURL              string     `json:"stylometry_sample_image_url"`
	StylometrySampleVideoURL              string     `json:"stylometry_sample_video_url"`
	StylometrySampleAverageSentenceLength float64    `json:"stylometry_sample_average_sentence_length"`
	StylometrySampleConjunctionFrequency  float64    `json:"stylometry_sample_conjunction_frequency"`
	StylometrySampleLikesCount            int        `json:"stylometry_sample_likes_count"`
	StylometrySampleIsCurrentUserCreator  int        `json:"stylometry_sample_is_current_user_creator"`
	StylometrySampleCreatedAt             *time.Time `json:"stylometry_sample_created_at,omitempty"`
	StylometrySampleUpdatedAt             *time.Time `json:"stylometry_sample_updated_at,omitempty"`
}

func SampleResponse(stylometry_sample model.StylometrySampleTextSample, stylometry_current_user_id uint) StylometrySampleResponse {
	stylometry_is_creator := 0
	if stylometry_sample.StylometrySampleCreatorID == stylometry_current_user_id {
		stylometry_is_creator = 1
	}

	return StylometrySampleResponse{
		StylometrySampleID:                    stylometry_sample.StylometrySampleID,
		StylometrySampleTextCreator:           stylometry_sample.StylometrySampleTextCreator,
		StylometrySampleDescr:                 stylometry_sample.StylometrySampleDescr,
		StylometrySampleImageURL:              stylometry_sample.StylometrySampleImageURL,
		StylometrySampleVideoURL:              stylometry_sample.StylometrySampleVideoURL,
		StylometrySampleAverageSentenceLength: stylometry_sample.StylometrySampleAverageSentenceLength,
		StylometrySampleConjunctionFrequency:  stylometry_sample.StylometrySampleConjunctionFrequency,
		StylometrySampleLikesCount:            len(stylometry_sample.StylometrySampleLikes),
		StylometrySampleIsCurrentUserCreator:  stylometry_is_creator,
		StylometrySampleCreatedAt:             stylometry_sample.StylometrySampleCreatedAt,
		StylometrySampleUpdatedAt:             stylometry_sample.StylometrySampleUpdatedAt,
	}
}

func SampleListResponse(stylometry_samples []model.StylometrySampleTextSample, stylometry_current_user_id uint) []StylometrySampleResponse {
	stylometry_response := make([]StylometrySampleResponse, 0, len(stylometry_samples))

	for _, stylometry_sample := range stylometry_samples {
		stylometry_response = append(stylometry_response, SampleResponse(stylometry_sample, stylometry_current_user_id))
	}

	return stylometry_response
}

type StylometrySampleUserResponse struct {
	StylometrySampleID   uint   `json:"stylometry_sample_id"`
	StylometrySampleName string `json:"stylometry_sample_name"`
}

func UserResponse(stylometry_user model.StylometrySampleUser) StylometrySampleUserResponse {
	return StylometrySampleUserResponse{
		StylometrySampleID:   stylometry_user.StylometrySampleID,
		StylometrySampleName: stylometry_user.StylometrySampleName,
	}
}
