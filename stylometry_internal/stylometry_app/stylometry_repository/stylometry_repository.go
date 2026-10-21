package repository

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"stylometry/stylometry_internal/stylometry_app/stylometry_analysis"
	"stylometry/stylometry_internal/stylometry_app/stylometry_model"
	"stylometry/stylometry_internal/stylometry_app/stylometry_password"
)

const DefaultImagePath = "/stylometry_static/stylometry_default_image.svg"

const DefaultImageURL = DefaultImagePath + "?v=gray-buttons-1"

const DefaultVideoURL = "/stylometry_static/stylometry_default_video.svg"

const stylometryDraftBrokenImageURL = "http://localhost:9000/stylometry/stylometry_draft_missing.jpg"

const stylometryDraftBrokenVideoURL = "http://localhost:9000/stylometry/stylometry_draft_missing.mp4"

var stylometryMediaClient = http.Client{Timeout: 500 * time.Millisecond}

// Repository хранит подключение к БД.
type Repository struct {
	DB *gorm.DB
}

// NewRepository создаёт репозиторий.
func NewRepository(stylometry_db *gorm.DB) *Repository {
	return &Repository{DB: stylometry_db}
}

// PublishedSamples возвращает только опубликованные карточки.
func (stylometry_repository *Repository) PublishedSamples() ([]model.Sample, error) {
	var stylometry_samples []model.Sample
	stylometry_err := stylometry_repository.DB.Preload("Likes").Where("\"Status\" = ?", model.StatusPublished).Order("\"ID\" ASC").Find(&stylometry_samples).Error
	if stylometry_err != nil {
		return nil, stylometry_err
	}
	stylometry_repository.fillDefaultMedia(stylometry_samples)
	return stylometry_samples, nil
}

// FeedSample возвращает одну опубликованную карточку для ленты.
// Без id берёт первую опубликованную запись.
// С id и next=true берёт следующую опубликованную запись после этого id.
func (stylometry_repository *Repository) FeedSample(stylometry_id *uint, stylometry_next bool) (model.Sample, bool, error) {
	var stylometry_sample model.Sample

	stylometry_query := stylometry_repository.DB.
		Preload("Likes").
		Where("\"Status\" = ?", model.StatusPublished)

	if stylometry_id != nil && stylometry_next {
		stylometry_query = stylometry_query.Where("\"ID\" > ?", *stylometry_id)
	}

	if stylometry_id != nil && !stylometry_next {
		stylometry_query = stylometry_query.Where("\"ID\" = ?", *stylometry_id)
	}

	stylometry_err := stylometry_query.
		Order("\"ID\" ASC").
		First(&stylometry_sample).Error

	if errors.Is(stylometry_err, gorm.ErrRecordNotFound) && stylometry_id != nil && stylometry_next {
		stylometry_err = stylometry_repository.DB.
			Preload("Likes").
			Where("\"Status\" = ?", model.StatusPublished).
			Order("\"ID\" ASC").
			First(&stylometry_sample).Error
	}

	if errors.Is(stylometry_err, gorm.ErrRecordNotFound) {
		return model.Sample{}, false, nil
	}
	if stylometry_err != nil {
		return model.Sample{}, false, stylometry_err
	}

	stylometry_repository.fillDefaultMediaForSample(&stylometry_sample)
	return stylometry_sample, true, nil
}

// SampleByID возвращает опубликованную карточку по ID.
func (stylometry_repository *Repository) SampleByID(stylometry_id uint) (model.Sample, bool, error) {
	var stylometry_sample model.Sample
	stylometry_err := stylometry_repository.DB.Preload("Likes").Where("\"ID\" = ? AND \"Status\" = ?", stylometry_id, model.StatusPublished).First(&stylometry_sample).Error
	if errors.Is(stylometry_err, gorm.ErrRecordNotFound) {
		return model.Sample{}, false, nil
	}
	if stylometry_err != nil {
		return model.Sample{}, false, stylometry_err
	}
	stylometry_repository.fillDefaultMediaForSample(&stylometry_sample)
	return stylometry_sample, true, nil
}

// DraftSample возвращает черновик текущего пользователя.
func (stylometry_repository *Repository) DraftSample(stylometry_user_id uint) (model.Sample, bool, error) {
	var stylometry_sample model.Sample
	stylometry_err := stylometry_repository.DB.Preload("Likes").Where("\"CreatorID\" = ? AND \"Status\" = ?", stylometry_user_id, model.StatusDraft).First(&stylometry_sample).Error
	if errors.Is(stylometry_err, gorm.ErrRecordNotFound) {
		return model.Sample{}, false, nil
	}
	if stylometry_err != nil {
		return model.Sample{}, false, stylometry_err
	}
	stylometry_repository.fillDefaultMediaForSample(&stylometry_sample)
	return stylometry_sample, true, nil
}

// CreateDraft создаёт или обновляет черновик при нажатии кнопки "Далее".
func (stylometry_repository *Repository) CreateDraft(stylometry_user_id uint, stylometry_text_creator string, stylometry_text string, stylometry_image_url string, stylometry_video_url string) (model.Sample, error) {
	stylometry_existing, stylometry_found, stylometry_err := stylometry_repository.DraftSample(stylometry_user_id)
	if stylometry_err != nil {
		return model.Sample{}, stylometry_err
	}

	stylometry_text_creator = strings.TrimSpace(stylometry_text_creator)
	if stylometry_text_creator == "" {
		return model.Sample{}, errors.New("автор текстового образца не заполнен")
	}
	if strings.TrimSpace(stylometry_text) == "" {
		return model.Sample{}, errors.New("текстовый отрывок не заполнен")
	}

	stylometry_metrics := analysis.AnalyzeText(stylometry_text)
	stylometry_image_url = stylometryNormalizeDraftMediaURL(stylometry_image_url, stylometryDraftBrokenImageURL)
	stylometry_video_url = stylometryNormalizeDraftMediaURL(stylometry_video_url, stylometryDraftBrokenVideoURL)

	if stylometry_found {
		stylometry_updates := map[string]interface{}{
			"TextCreator":           stylometry_text_creator,
			"ShortDescription":      stylometry_text,
			"ImageURL":              stylometry_image_url,
			"VideoURL":              stylometry_video_url,
			"AverageSentenceLength": stylometry_metrics.AverageSentenceLength,
			"ConjunctionFrequency":  stylometry_metrics.ConjunctionFrequency,
		}
		stylometry_err = stylometry_repository.DB.Model(&model.Sample{}).
			Where("\"ID\" = ? AND \"CreatorID\" = ?", stylometry_existing.ID, stylometry_user_id).
			Updates(stylometry_updates).Error
		if stylometry_err != nil {
			return model.Sample{}, stylometry_err
		}
		stylometry_existing.TextCreator = stylometry_text_creator
		stylometry_existing.ShortDescription = stylometry_text
		stylometry_existing.ImageURL = stylometry_image_url
		stylometry_existing.VideoURL = stylometry_video_url
		stylometry_existing.AverageSentenceLength = stylometry_metrics.AverageSentenceLength
		stylometry_existing.ConjunctionFrequency = stylometry_metrics.ConjunctionFrequency
		stylometry_repository.fillDefaultMediaForSample(&stylometry_existing)
		return stylometry_existing, nil
	}

	stylometry_sample := model.Sample{
		TextCreator:           stylometry_text_creator,
		ShortDescription:      stylometry_text,
		Status:                model.StatusDraft,
		ImageURL:              stylometry_image_url,
		VideoURL:              stylometry_video_url,
		AverageSentenceLength: stylometry_metrics.AverageSentenceLength,
		ConjunctionFrequency:  stylometry_metrics.ConjunctionFrequency,
		CreatorID:             stylometry_user_id,
	}
	stylometry_err = stylometry_repository.DB.Create(&stylometry_sample).Error
	if stylometry_err != nil {
		return model.Sample{}, stylometry_err
	}
	stylometry_repository.fillDefaultMediaForSample(&stylometry_sample)
	return stylometry_sample, nil
}

// PublishDraft публикует черновик текущего пользователя.
func (stylometry_repository *Repository) PublishDraft(stylometry_user_id uint, stylometry_description string, stylometry_avg_len float64, stylometry_conj_freq float64) error {
	stylometry_updated_at := time.Now()
	stylometry_result := stylometry_repository.DB.Model(&model.Sample{}).Where("\"CreatorID\" = ? AND \"Status\" = ?", stylometry_user_id, model.StatusDraft).Updates(map[string]interface{}{"ShortDescription": stylometry_description, "AverageSentenceLength": stylometry_avg_len, "ConjunctionFrequency": stylometry_conj_freq, "Status": model.StatusPublished, "UpdatedAt": &stylometry_updated_at})
	if stylometry_result.Error != nil {
		return stylometry_result.Error
	}
	if stylometry_result.RowsAffected == 0 {
		return errors.New("черновик для публикации не найден")
	}
	return nil
}

// SetLike ставит или снимает лайк текущего пользователя у опубликованной карточки.
func (stylometry_repository *Repository) SetLike(stylometry_user_id uint, stylometry_id uint, stylometry_like_value int) (bool, error) {
	var stylometry_count int64
	stylometry_err := stylometry_repository.DB.Model(&model.Sample{}).
		Where("\"ID\" = ? AND \"Status\" = ?", stylometry_id, model.StatusPublished).
		Count(&stylometry_count).Error
	if stylometry_err != nil {
		return false, stylometry_err
	}
	if stylometry_count == 0 {
		return false, nil
	}

	stylometry_like := model.Like{
		UserID:   stylometry_user_id,
		SampleID: stylometry_id,
	}

	if stylometry_like_value == 1 {
		stylometry_err = stylometry_repository.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&stylometry_like).Error
		return true, stylometry_err
	}

	stylometry_err = stylometry_repository.DB.
		Where("\"UserID\" = ? AND \"SampleID\" = ?", stylometry_user_id, stylometry_id).
		Delete(&model.Like{}).Error
	return true, stylometry_err
}

// RegisterUser создаёт пользователя с bcrypt-хешем пароля.
func (stylometry_repository *Repository) RegisterUser(
	stylometry_name string,
	stylometry_raw_password string,
) (model.User, error) {
	stylometry_hash, stylometry_err := password.Hash(stylometry_raw_password)
	if stylometry_err != nil {
		return model.User{}, stylometry_err
	}

	stylometry_user := model.User{
		Name:     stylometry_name,
		Password: stylometry_hash,
	}

	if stylometry_err := stylometry_repository.DB.Create(&stylometry_user).Error; stylometry_err != nil {
		return model.User{}, stylometry_err
	}

	return stylometry_user, nil
}

// UserByName находит пользователя для будущей проверки пароля при входе.
func (stylometry_repository *Repository) UserByName(
	stylometry_name string,
) (model.User, error) {
	var stylometry_user model.User

	stylometry_err := stylometry_repository.DB.
		Where(`"Name" = ?`, stylometry_name).
		First(&stylometry_user).Error

	return stylometry_user, stylometry_err
}

// UserExists подтверждает, что sub токена указывает на существующего пользователя.
func (stylometry_repository *Repository) UserExists(stylometry_user_id uint) (bool, error) {
	var stylometry_count int64
	stylometry_err := stylometry_repository.DB.Model(&model.User{}).
		Where(`"ID" = ?`, stylometry_user_id).
		Count(&stylometry_count).Error
	return stylometry_count == 1, stylometry_err
}

// SampleOwnerID находит владельца существующей неудалённой карточки.
func (stylometry_repository *Repository) SampleOwnerID(stylometry_id uint) (uint, bool, error) {
	var stylometry_sample model.Sample
	stylometry_err := stylometry_repository.DB.Select("CreatorID").
		Where(`"ID" = ? AND "Status" <> ?`, stylometry_id, model.StatusDeleted).
		First(&stylometry_sample).Error
	if errors.Is(stylometry_err, gorm.ErrRecordNotFound) {
		return 0, false, nil
	}
	if stylometry_err != nil {
		return 0, false, stylometry_err
	}
	return stylometry_sample.CreatorID, true, nil
}

// DeleteSampleSQL логически удаляет карточку через явный курсор PostgreSQL.
func (stylometry_repository *Repository) DeleteSampleSQL(stylometry_user_id uint, stylometry_id uint) (stylometry_err error) {
	stylometry_sql_db, stylometry_err := stylometry_repository.DB.DB()
	if stylometry_err != nil {
		return stylometry_err
	}

	stylometry_context := context.Background()
	stylometry_tx, stylometry_err := stylometry_sql_db.BeginTx(stylometry_context, nil)
	if stylometry_err != nil {
		return stylometry_err
	}

	stylometry_committed := false
	stylometry_cursor_open := false
	defer func() {
		if stylometry_cursor_open {
			if _, stylometry_close_err := stylometry_tx.ExecContext(stylometry_context, "CLOSE stylometry_sample_delete_cursor"); stylometry_err == nil && stylometry_close_err != nil {
				stylometry_err = stylometry_close_err
			}
		}
		if !stylometry_committed {
			_ = stylometry_tx.Rollback()
		}
	}()

	// Курсор выбирает и блокирует именно доступную текущему пользователю карточку.
	_, stylometry_err = stylometry_tx.ExecContext(stylometry_context,
		"DECLARE stylometry_sample_delete_cursor NO SCROLL CURSOR FOR SELECT \"ID\" FROM \"StylometrySamples\" WHERE \"ID\" = $1 AND \"Status\" <> $2 AND \"CreatorID\" = $3 FOR UPDATE",
		stylometry_id, string(model.StatusDeleted), stylometry_user_id)
	if stylometry_err != nil {
		return stylometry_err
	}
	stylometry_cursor_open = true

	var stylometry_deleted_id uint
	stylometry_err = stylometry_tx.QueryRowContext(stylometry_context, "FETCH NEXT FROM stylometry_sample_delete_cursor").Scan(&stylometry_deleted_id)
	if errors.Is(stylometry_err, sql.ErrNoRows) {
		return errors.New("карточка для удаления не найдена")
	}
	if stylometry_err != nil {
		return stylometry_err
	}

	stylometry_result, stylometry_err := stylometry_tx.ExecContext(stylometry_context,
		"UPDATE \"StylometrySamples\" SET \"Status\" = $1 WHERE \"ID\" = $2 AND \"Status\" <> $3 AND \"CreatorID\" = $4",
		string(model.StatusDeleted), stylometry_deleted_id, string(model.StatusDeleted), stylometry_user_id)
	if stylometry_err != nil {
		return stylometry_err
	}
	stylometry_rows_affected, stylometry_err := stylometry_result.RowsAffected()
	if stylometry_err != nil {
		return stylometry_err
	}
	if stylometry_rows_affected != 1 {
		return errors.New("карточка для удаления не найдена")
	}

	if _, stylometry_err = stylometry_tx.ExecContext(stylometry_context, "CLOSE stylometry_sample_delete_cursor"); stylometry_err != nil {
		return stylometry_err
	}
	stylometry_cursor_open = false
	if stylometry_err = stylometry_tx.Commit(); stylometry_err != nil {
		return stylometry_err
	}
	stylometry_committed = true
	return nil
}

// FilteredSamples возвращает опубликованные карточки с фильтром по длине.
func (stylometry_repository *Repository) FilteredSamples(stylometry_max_avg_len *float64) ([]model.Sample, error) {
	var stylometry_samples []model.Sample
	stylometry_query := stylometry_repository.DB.Preload("Likes").Where("\"Status\" = ?", model.StatusPublished)
	if stylometry_max_avg_len != nil {
		stylometry_query = stylometry_query.Where("\"AverageSentenceLength\" <= ?", *stylometry_max_avg_len)
	}
	stylometry_err := stylometry_query.Order("\"ID\" ASC").Find(&stylometry_samples).Error
	if stylometry_err != nil {
		return nil, stylometry_err
	}
	stylometry_repository.fillDefaultMedia(stylometry_samples)
	return stylometry_samples, nil
}

// fillDefaultMedia ставит дефолтные ссылки, если в БД они пустые.
func (stylometry_repository *Repository) fillDefaultMedia(stylometry_samples []model.Sample) {
	for stylometry_index := range stylometry_samples {
		stylometry_repository.fillDefaultMediaForSample(&stylometry_samples[stylometry_index])
	}
}

// fillDefaultMediaForSample ставит дефолтные ссылки в одну карточку.
func (stylometry_repository *Repository) fillDefaultMediaForSample(stylometry_sample *model.Sample) {
	if strings.TrimSpace(stylometry_sample.ImageURL) == DefaultImagePath {
		stylometry_sample.ImageURL = DefaultImageURL
	} else if !stylometryMediaAvailable(stylometry_sample.ImageURL) {
		stylometry_sample.ImageURL = DefaultImageURL
	}
	if !stylometryMediaAvailable(stylometry_sample.VideoURL) {
		stylometry_sample.VideoURL = DefaultVideoURL
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
