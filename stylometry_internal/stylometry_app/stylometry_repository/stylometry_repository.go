package repository

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"stylometry/stylometry_internal/stylometry_app/stylometry_analysis"
	"stylometry/stylometry_internal/stylometry_app/stylometry_model"
	"stylometry/stylometry_internal/stylometry_app/stylometry_session"
)

const DefaultImagePath = "/stylometry_static/stylometry_default_image.svg"

const DefaultImageURL = DefaultImagePath + "?v=gray-buttons-1"

const DefaultVideoURL = "/stylometry_static/stylometry_default_video.svg"

const stylometryDraftBrokenImageURL = "http://localhost:9000/stylometry/stylometry_draft_missing.jpg"

const stylometryDraftBrokenVideoURL = "http://localhost:9000/stylometry/stylometry_draft_missing.mp4"

var stylometryMediaClient = http.Client{Timeout: 500 * time.Millisecond}

// StylometrySampleRepository хранит подключение к БД.
type StylometrySampleRepository struct {
	DB *gorm.DB
}

// NewStylometrySampleRepository создаёт репозиторий.
func NewStylometrySampleRepository(stylometry_db *gorm.DB) *StylometrySampleRepository {
	return &StylometrySampleRepository{DB: stylometry_db}
}

func stylometryCurrentUserID() uint {
	return stylometry_session.CurrentUser().StylometrySampleID
}

// PublishedSamples возвращает только опубликованные карточки.
func (stylometry_repository *StylometrySampleRepository) PublishedSamples() ([]model.StylometrySampleTextSample, error) {
	var stylometry_samples []model.StylometrySampleTextSample
	stylometry_err := stylometry_repository.DB.Preload("StylometrySampleLikes").Where("\"StylometrySampleStatus\" = ?", model.StatusPublished).Order("\"StylometrySampleID\" ASC").Find(&stylometry_samples).Error
	if stylometry_err != nil {
		return nil, stylometry_err
	}
	stylometry_repository.fillDefaultMedia(stylometry_samples)
	return stylometry_samples, nil
}

// FeedSample возвращает одну опубликованную карточку для ленты.
// Без id берёт первую опубликованную запись.
// С id и next=true берёт следующую опубликованную запись после этого id.
func (stylometry_repository *StylometrySampleRepository) FeedSample(stylometry_id *uint, stylometry_next bool) (model.StylometrySampleTextSample, bool, error) {
	var stylometry_sample model.StylometrySampleTextSample

	stylometry_query := stylometry_repository.DB.
		Preload("StylometrySampleLikes").
		Where("\"StylometrySampleStatus\" = ?", model.StatusPublished)

	if stylometry_id != nil && stylometry_next {
		stylometry_query = stylometry_query.Where("\"StylometrySampleID\" > ?", *stylometry_id)
	}

	if stylometry_id != nil && !stylometry_next {
		stylometry_query = stylometry_query.Where("\"StylometrySampleID\" = ?", *stylometry_id)
	}

	stylometry_err := stylometry_query.
		Order("\"StylometrySampleID\" ASC").
		First(&stylometry_sample).Error

	if errors.Is(stylometry_err, gorm.ErrRecordNotFound) && stylometry_id != nil && stylometry_next {
		stylometry_err = stylometry_repository.DB.
			Preload("StylometrySampleLikes").
			Where("\"StylometrySampleStatus\" = ?", model.StatusPublished).
			Order("\"StylometrySampleID\" ASC").
			First(&stylometry_sample).Error
	}

	if errors.Is(stylometry_err, gorm.ErrRecordNotFound) {
		return model.StylometrySampleTextSample{}, false, nil
	}
	if stylometry_err != nil {
		return model.StylometrySampleTextSample{}, false, stylometry_err
	}

	stylometry_repository.fillDefaultMediaForSample(&stylometry_sample)
	return stylometry_sample, true, nil
}

// SampleByID возвращает опубликованную карточку по ID.
func (stylometry_repository *StylometrySampleRepository) SampleByID(stylometry_id uint) (model.StylometrySampleTextSample, bool, error) {
	var stylometry_sample model.StylometrySampleTextSample
	stylometry_err := stylometry_repository.DB.Preload("StylometrySampleLikes").Where("\"StylometrySampleID\" = ? AND \"StylometrySampleStatus\" = ?", stylometry_id, model.StatusPublished).First(&stylometry_sample).Error
	if errors.Is(stylometry_err, gorm.ErrRecordNotFound) {
		return model.StylometrySampleTextSample{}, false, nil
	}
	if stylometry_err != nil {
		return model.StylometrySampleTextSample{}, false, stylometry_err
	}
	stylometry_repository.fillDefaultMediaForSample(&stylometry_sample)
	return stylometry_sample, true, nil
}

// DraftSample возвращает черновик текущего пользователя.
func (stylometry_repository *StylometrySampleRepository) DraftSample() (model.StylometrySampleTextSample, bool, error) {
	var stylometry_sample model.StylometrySampleTextSample
	stylometry_err := stylometry_repository.DB.Preload("StylometrySampleLikes").Where("\"StylometrySampleCreatorID\" = ? AND \"StylometrySampleStatus\" = ?", stylometryCurrentUserID(), model.StatusDraft).First(&stylometry_sample).Error
	if errors.Is(stylometry_err, gorm.ErrRecordNotFound) {
		return model.StylometrySampleTextSample{}, false, nil
	}
	if stylometry_err != nil {
		return model.StylometrySampleTextSample{}, false, stylometry_err
	}
	stylometry_repository.fillDefaultMediaForSample(&stylometry_sample)
	return stylometry_sample, true, nil
}

// CreateDraft создаёт или обновляет черновик при нажатии кнопки "Далее".
func (stylometry_repository *StylometrySampleRepository) CreateDraft(stylometry_text_creator string, stylometry_text string, stylometry_image_url string, stylometry_video_url string) (model.StylometrySampleTextSample, error) {
	stylometry_existing, stylometry_found, stylometry_err := stylometry_repository.DraftSample()
	if stylometry_err != nil {
		return model.StylometrySampleTextSample{}, stylometry_err
	}

	stylometry_text = strings.TrimSpace(stylometry_text)
	if stylometry_text == "" {
		return model.StylometrySampleTextSample{}, errors.New("текстовый отрывок не заполнен")
	}

	stylometry_metrics := analysis.AnalyzeText(stylometry_text)
	stylometry_image_url = stylometryNormalizeDraftMediaURL(stylometry_image_url, stylometryDraftBrokenImageURL)
	stylometry_video_url = stylometryNormalizeDraftMediaURL(stylometry_video_url, stylometryDraftBrokenVideoURL)

	if stylometry_found {
		stylometry_updates := map[string]interface{}{
			"StylometrySampleTextCreator":           stylometry_text_creator,
			"StylometrySampleShortDescription":      stylometry_text,
			"StylometrySampleImageURL":              stylometry_image_url,
			"StylometrySampleVideoURL":              stylometry_video_url,
			"StylometrySampleAverageSentenceLength": stylometry_metrics.StylometrySampleAverageSentenceLength,
			"StylometrySampleConjunctionFrequency":  stylometry_metrics.StylometrySampleConjunctionFrequency,
		}
		stylometry_err = stylometry_repository.DB.Model(&model.StylometrySampleTextSample{}).
			Where("\"StylometrySampleID\" = ?", stylometry_existing.StylometrySampleID).
			Updates(stylometry_updates).Error
		if stylometry_err != nil {
			return model.StylometrySampleTextSample{}, stylometry_err
		}
		stylometry_existing.StylometrySampleTextCreator = stylometry_text_creator
		stylometry_existing.StylometrySampleDescr = stylometry_text
		stylometry_existing.StylometrySampleImageURL = stylometry_image_url
		stylometry_existing.StylometrySampleVideoURL = stylometry_video_url
		stylometry_existing.StylometrySampleAverageSentenceLength = stylometry_metrics.StylometrySampleAverageSentenceLength
		stylometry_existing.StylometrySampleConjunctionFrequency = stylometry_metrics.StylometrySampleConjunctionFrequency
		stylometry_repository.fillDefaultMediaForSample(&stylometry_existing)
		return stylometry_existing, nil
	}

	stylometry_sample := model.StylometrySampleTextSample{
		StylometrySampleTextCreator:           stylometry_text_creator,
		StylometrySampleDescr:                 stylometry_text,
		StylometrySampleStatus:                model.StatusDraft,
		StylometrySampleImageURL:              stylometry_image_url,
		StylometrySampleVideoURL:              stylometry_video_url,
		StylometrySampleAverageSentenceLength: stylometry_metrics.StylometrySampleAverageSentenceLength,
		StylometrySampleConjunctionFrequency:  stylometry_metrics.StylometrySampleConjunctionFrequency,
		StylometrySampleCreatorID:             stylometryCurrentUserID(),
	}
	stylometry_err = stylometry_repository.DB.Create(&stylometry_sample).Error
	if stylometry_err != nil {
		return model.StylometrySampleTextSample{}, stylometry_err
	}
	stylometry_repository.fillDefaultMediaForSample(&stylometry_sample)
	return stylometry_sample, nil
}

// PublishDraft публикует черновик текущего пользователя.
func (stylometry_repository *StylometrySampleRepository) PublishDraft(stylometry_description string, stylometry_avg_len float64, stylometry_conj_freq float64) error {
	stylometry_updated_at := time.Now()
	stylometry_result := stylometry_repository.DB.Model(&model.StylometrySampleTextSample{}).Where("\"StylometrySampleCreatorID\" = ? AND \"StylometrySampleStatus\" = ?", stylometryCurrentUserID(), model.StatusDraft).Updates(map[string]interface{}{"StylometrySampleShortDescription": stylometry_description, "StylometrySampleAverageSentenceLength": stylometry_avg_len, "StylometrySampleConjunctionFrequency": stylometry_conj_freq, "StylometrySampleStatus": model.StatusPublished, "StylometrySampleUpdatedAt": &stylometry_updated_at})
	if stylometry_result.Error != nil {
		return stylometry_result.Error
	}
	if stylometry_result.RowsAffected == 0 {
		return errors.New("черновик для публикации не найден")
	}
	return nil
}

// SetLike ставит или снимает лайк текущего пользователя у опубликованной карточки.
func (stylometry_repository *StylometrySampleRepository) SetLike(stylometry_id uint, stylometry_like_value int) (bool, error) {
	var stylometry_count int64
	stylometry_err := stylometry_repository.DB.Model(&model.StylometrySampleTextSample{}).
		Where("\"StylometrySampleID\" = ? AND \"StylometrySampleStatus\" = ?", stylometry_id, model.StatusPublished).
		Count(&stylometry_count).Error
	if stylometry_err != nil {
		return false, stylometry_err
	}
	if stylometry_count == 0 {
		return false, nil
	}

	stylometry_like := model.StylometrySampleLike{
		StylometrySampleUserID:       stylometryCurrentUserID(),
		StylometrySampleTextSampleID: stylometry_id,
	}

	if stylometry_like_value == 1 {
		stylometry_err = stylometry_repository.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&stylometry_like).Error
		return true, stylometry_err
	}

	stylometry_err = stylometry_repository.DB.
		Where("\"StylometrySampleUserID\" = ? AND \"StylometrySampleTextSampleID\" = ?", stylometryCurrentUserID(), stylometry_id).
		Delete(&model.StylometrySampleLike{}).Error
	return true, stylometry_err
}

// RegisterUser создаёт нового пользователя для показа пользовательского домена API.
func (stylometry_repository *StylometrySampleRepository) RegisterUser(stylometry_name string, stylometry_password string) (model.StylometrySampleUser, error) {
	stylometry_user := model.StylometrySampleUser{
		StylometrySampleName:     stylometry_name,
		StylometrySamplePassword: stylometry_password,
	}
	stylometry_err := stylometry_repository.DB.Create(&stylometry_user).Error
	if stylometry_err != nil {
		return model.StylometrySampleUser{}, stylometry_err
	}
	return stylometry_user, nil
}

// DeleteSampleSQL логически удаляет карточку через SQL UPDATE без ORM-модели.
func (stylometry_repository *StylometrySampleRepository) DeleteSampleSQL(stylometry_id uint) error {
	stylometry_sql_db, stylometry_err := stylometry_repository.DB.DB()
	if stylometry_err != nil {
		return stylometry_err
	}

	stylometry_rows, stylometry_err := stylometry_sql_db.QueryContext(context.Background(),
		"UPDATE \"StylometrySampleTextSamples\" SET \"StylometrySampleStatus\" = $1 WHERE \"StylometrySampleID\" = $2 AND \"StylometrySampleStatus\" <> $3 AND \"StylometrySampleCreatorID\" = $4 RETURNING \"StylometrySampleID\"",
		string(model.StatusDeleted), stylometry_id, string(model.StatusDeleted), stylometryCurrentUserID())
	if stylometry_err != nil {
		return stylometry_err
	}
	defer stylometry_rows.Close()

	if stylometry_rows.Next() {
		var stylometry_deleted_id uint
		if stylometry_err := stylometry_rows.Scan(&stylometry_deleted_id); stylometry_err != nil {
			return stylometry_err
		}
		return stylometry_rows.Err()
	}
	if stylometry_err := stylometry_rows.Err(); stylometry_err != nil {
		return stylometry_err
	}
	return errors.New("карточка для удаления не найдена")
}

// FilteredSamples возвращает опубликованные карточки с фильтром по длине.
func (stylometry_repository *StylometrySampleRepository) FilteredSamples(stylometry_max_avg_len *float64) ([]model.StylometrySampleTextSample, error) {
	var stylometry_samples []model.StylometrySampleTextSample
	stylometry_query := stylometry_repository.DB.Preload("StylometrySampleLikes").Where("\"StylometrySampleStatus\" = ?", model.StatusPublished)
	if stylometry_max_avg_len != nil {
		stylometry_query = stylometry_query.Where("\"StylometrySampleAverageSentenceLength\" <= ?", *stylometry_max_avg_len)
	}
	stylometry_err := stylometry_query.Order("\"StylometrySampleID\" ASC").Find(&stylometry_samples).Error
	if stylometry_err != nil {
		return nil, stylometry_err
	}
	stylometry_repository.fillDefaultMedia(stylometry_samples)
	return stylometry_samples, nil
}

// fillDefaultMedia ставит дефолтные ссылки, если в БД они пустые.
func (stylometry_repository *StylometrySampleRepository) fillDefaultMedia(stylometry_samples []model.StylometrySampleTextSample) {
	for stylometry_index := range stylometry_samples {
		stylometry_repository.fillDefaultMediaForSample(&stylometry_samples[stylometry_index])
	}
}

// fillDefaultMediaForSample ставит дефолтные ссылки в одну карточку.
func (stylometry_repository *StylometrySampleRepository) fillDefaultMediaForSample(stylometry_sample *model.StylometrySampleTextSample) {
	if strings.TrimSpace(stylometry_sample.StylometrySampleImageURL) == DefaultImagePath {
		stylometry_sample.StylometrySampleImageURL = DefaultImageURL
	} else if !stylometryMediaAvailable(stylometry_sample.StylometrySampleImageURL) {
		stylometry_sample.StylometrySampleImageURL = DefaultImageURL
	}
	if !stylometryMediaAvailable(stylometry_sample.StylometrySampleVideoURL) {
		stylometry_sample.StylometrySampleVideoURL = DefaultVideoURL
	}
}

func stylometryNormalizeDraftMediaURL(stylometry_value string, stylometry_fallback string) string {
	stylometry_value = strings.TrimSpace(stylometry_value)
	if stylometry_value == "" {
		return stylometry_fallback
	}
	return stylometry_value
}

func stylometryMediaAvailable(stylometry_raw_url string) bool {
	stylometry_raw_url = strings.TrimSpace(stylometry_raw_url)
	if stylometry_raw_url == "" {
		return false
	}
	if strings.HasPrefix(stylometry_raw_url, "/") {
		return true
	}
	if !strings.HasPrefix(stylometry_raw_url, "http://") && !strings.HasPrefix(stylometry_raw_url, "https://") {
		return true
	}

	stylometry_check_url := stylometryInternalMediaURL(stylometry_raw_url)
	stylometry_request, stylometry_err := http.NewRequest(http.MethodHead, stylometry_check_url, nil)
	if stylometry_err != nil {
		return false
	}

	stylometry_response, stylometry_err := stylometryMediaClient.Do(stylometry_request)
	if stylometry_err != nil {
		return false
	}
	defer stylometry_response.Body.Close()

	return stylometry_response.StatusCode >= http.StatusOK && stylometry_response.StatusCode < http.StatusBadRequest
}

func stylometryInternalMediaURL(stylometry_raw_url string) string {
	stylometry_internal_url := strings.TrimRight(os.Getenv("MINIO_INTERNAL_URL"), "/")
	if stylometry_internal_url == "" {
		return stylometry_raw_url
	}

	for _, stylometry_public_url := range []string{
		strings.TrimRight(os.Getenv("MINIO_PUBLIC_URL"), "/"),
		"http://localhost:9000",
		"http://127.0.0.1:9000",
	} {
		if stylometry_public_url != "" && strings.HasPrefix(stylometry_raw_url, stylometry_public_url) {
			return stylometry_internal_url + strings.TrimPrefix(stylometry_raw_url, stylometry_public_url)
		}
	}

	return stylometry_raw_url
}
