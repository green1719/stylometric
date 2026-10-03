package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"stylometry/stylometry_internal/stylometry_app/stylometry_analysis"
	"stylometry/stylometry_internal/stylometry_app/stylometry_model"
)

const DefaultUserID uint = 1

const DefaultImageURL = "/stylometry_static/stylometry_default_image.svg"

const DefaultVideoURL = "/stylometry_static/stylometry_default_video.svg"

// StylometrySampleRepository хранит подключение к БД.
type StylometrySampleRepository struct {
	DB *gorm.DB
}

// NewStylometrySampleRepository создаёт репозиторий.
func NewStylometrySampleRepository(stylometry_db *gorm.DB) *StylometrySampleRepository {
	return &StylometrySampleRepository{DB: stylometry_db}
}

// PublishedSamples возвращает только опубликованные карточки.
func (stylometry_repository *StylometrySampleRepository) PublishedSamples() ([]model.StylometrySampleTextSample, error) {
	var stylometry_samples []model.StylometrySampleTextSample
	stylometry_err := stylometry_repository.DB.Preload("StylometrySampleLikes").Where("status = ?", model.StatusPublished).Order("id ASC").Find(&stylometry_samples).Error
	if stylometry_err != nil {
		return nil, stylometry_err
	}
	stylometry_repository.fillDefaultMedia(stylometry_samples)
	return stylometry_samples, nil
}

// SampleByID возвращает опубликованную карточку по ID.
func (stylometry_repository *StylometrySampleRepository) SampleByID(stylometry_id uint) (model.StylometrySampleTextSample, bool, error) {
	var stylometry_sample model.StylometrySampleTextSample
	stylometry_err := stylometry_repository.DB.Preload("StylometrySampleLikes").Where("id = ? AND status = ?", stylometry_id, model.StatusPublished).First(&stylometry_sample).Error
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
	stylometry_err := stylometry_repository.DB.Preload("StylometrySampleLikes").Where("creator_id = ? AND status = ?", DefaultUserID, model.StatusDraft).First(&stylometry_sample).Error
	if errors.Is(stylometry_err, gorm.ErrRecordNotFound) {
		return model.StylometrySampleTextSample{}, false, nil
	}
	if stylometry_err != nil {
		return model.StylometrySampleTextSample{}, false, stylometry_err
	}
	stylometry_repository.fillDefaultMediaForSample(&stylometry_sample)
	return stylometry_sample, true, nil
}

// CreateDraft создаёт черновик при нажатии кнопки "Далее".
func (stylometry_repository *StylometrySampleRepository) CreateDraft(stylometry_title string, stylometry_fragment_key string, stylometry_fragment_text string) (model.StylometrySampleTextSample, error) {
	stylometry_existing, stylometry_found, stylometry_err := stylometry_repository.DraftSample()
	if stylometry_err != nil {
		return model.StylometrySampleTextSample{}, stylometry_err
	}
	if stylometry_found {
		return stylometry_existing, nil
	}

	stylometry_fragment := analysis.ResolveFragment(stylometry_title, stylometry_fragment_key, stylometry_fragment_text)
	stylometry_metrics := analysis.AnalyzeFragment(stylometry_fragment.StylometrySampleText)
	stylometry_author := stylometry_fragment.StylometrySampleAuthorName
	if stylometry_author == "" {
		stylometry_author = "Не указан"
	}

	stylometry_sample := model.StylometrySampleTextSample{
		StylometrySampleTitle:      stylometry_title,
		StylometrySampleDescr:      stylometry_fragment.StylometrySampleText,
		StylometrySampleStatus:     model.StatusDraft,
		StylometrySampleAuthorName: stylometry_author,
		StylometrySampleAvgSentLen: stylometry_metrics.StylometrySampleAvgSentLen,
		StylometrySampleConjFreq:   stylometry_metrics.StylometrySampleConjFreq,
		StylometrySampleCreatedAt:  time.Now(),
		StylometrySampleCreatorID:  DefaultUserID,
	}
	stylometry_err = stylometry_repository.DB.Create(&stylometry_sample).Error
	if stylometry_err != nil {
		return model.StylometrySampleTextSample{}, stylometry_err
	}
	stylometry_repository.fillDefaultMediaForSample(&stylometry_sample)
	return stylometry_sample, nil
}

// PublishDraft публикует черновик текущего пользователя.
func (stylometry_repository *StylometrySampleRepository) PublishDraft(stylometry_description string, stylometry_author string, stylometry_avg_len float64, stylometry_conj_freq float64) error {
	stylometry_now := time.Now()
	stylometry_result := stylometry_repository.DB.Model(&model.StylometrySampleTextSample{}).Where("creator_id = ? AND status = ?", DefaultUserID, model.StatusDraft).Updates(map[string]interface{}{"short_description": stylometry_description, "author_name": stylometry_author, "avg_sentence_length": stylometry_avg_len, "conjunction_frequency": stylometry_conj_freq, "status": model.StatusPublished, "formed_at": &stylometry_now})
	if stylometry_result.Error != nil {
		return stylometry_result.Error
	}
	if stylometry_result.RowsAffected == 0 {
		return errors.New("черновик для публикации не найден")
	}
	return nil
}

// DeleteSampleSQL логически удаляет карточку через SQL UPDATE без ORM-модели.
func (stylometry_repository *StylometrySampleRepository) DeleteSampleSQL(stylometry_id uint) error {
	stylometry_sql_db, stylometry_err := stylometry_repository.DB.DB()
	if stylometry_err != nil {
		return stylometry_err
	}

	stylometry_result, stylometry_err := stylometry_sql_db.ExecContext(context.Background(),
		"UPDATE stylometry_text_samples SET status = $1 WHERE id = $2 AND status <> $3",
		string(model.StatusDeleted), stylometry_id, string(model.StatusDeleted))
	if stylometry_err != nil {
		return stylometry_err
	}

	stylometry_rows_affected, stylometry_err := stylometry_result.RowsAffected()
	if stylometry_err != nil {
		return stylometry_err
	}
	if stylometry_rows_affected == 0 {
		return errors.New("карточка для удаления не найдена")
	}
	return nil
}

// FilteredSamples возвращает опубликованные карточки с поиском по названию и фильтром по длине.
func (stylometry_repository *StylometrySampleRepository) FilteredSamples(stylometry_search string, stylometry_max_avg_len *float64) ([]model.StylometrySampleTextSample, error) {
	var stylometry_samples []model.StylometrySampleTextSample
	stylometry_query := stylometry_repository.DB.Preload("StylometrySampleLikes").Where("status = ?", model.StatusPublished)
	if stylometry_search != "" {
		stylometry_query = stylometry_query.Where("lower(title) LIKE lower(?)", "%"+stylometry_search+"%")
	}
	if stylometry_max_avg_len != nil {
		stylometry_query = stylometry_query.Where("avg_sentence_length <= ?", *stylometry_max_avg_len)
	}
	stylometry_err := stylometry_query.Order("id ASC").Find(&stylometry_samples).Error
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
	if stylometry_sample.StylometrySampleImageURL == "" {
		stylometry_sample.StylometrySampleImageURL = DefaultImageURL
	}
	if stylometry_sample.StylometrySampleVideoURL == "" {
		stylometry_sample.StylometrySampleVideoURL = DefaultVideoURL
	}
}
