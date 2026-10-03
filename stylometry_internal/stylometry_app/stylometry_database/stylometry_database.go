package database

import (
	"fmt"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"stylometry/stylometry_internal/stylometry_app/stylometry_analysis"
	"stylometry/stylometry_internal/stylometry_app/stylometry_model"
)

// NewDatabase создаёт подключение к БД, миграции и стартовые данные.
func NewDatabase() (*gorm.DB, error) {
	var stylometry_db *gorm.DB
	var stylometry_err error
	for stylometry_attempt := 1; stylometry_attempt <= 10; stylometry_attempt++ {
		stylometry_db, stylometry_err = gorm.Open(postgres.Open(stylometryBuildDSN()), &gorm.Config{})
		if stylometry_err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if stylometry_err != nil {
		return nil, stylometry_err
	}

	stylometry_err = stylometry_db.AutoMigrate(
		&model.StylometrySampleUser{},
		&model.StylometrySampleTextSample{},
		&model.StylometrySampleLike{},
	)
	if stylometry_err != nil {
		return nil, stylometry_err
	}

	if stylometry_err = seedDatabase(stylometry_db); stylometry_err != nil {
		return nil, stylometry_err
	}

	return stylometry_db, nil
}

// stylometryBuildDSN собирает PostgreSQL DSN из переменных окружения.
func stylometryBuildDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		stylometryEnv("DB_HOST", "127.0.0.1"),
		stylometryEnv("DB_PORT", "5432"),
		stylometryEnv("DB_USER", "stylometry_user"),
		stylometryEnv("DB_PASS", "stylometry_password"),
		stylometryEnv("DB_NAME", "stylometry_db"),
	)
}

// stylometryEnv читает переменную окружения или отдаёт запасное значение.
func stylometryEnv(stylometry_key string, stylometry_fallback string) string {
	stylometry_value := os.Getenv(stylometry_key)
	if stylometry_value == "" {
		return stylometry_fallback
	}
	return stylometry_value
}

// seedDatabase создаёт стартовые данные для показа лабораторной.
func seedDatabase(stylometry_db *gorm.DB) error {
	stylometry_now := time.Now()

	stylometry_users := stylometrySeedUsers(stylometry_now)
	if stylometry_err := stylometry_db.Clauses(clause.OnConflict{DoNothing: true}).Create(&stylometry_users).Error; stylometry_err != nil {
		return stylometry_err
	}

	stylometry_samples := stylometrySeedSamples(stylometry_now)
	if stylometry_err := stylometry_db.Clauses(clause.OnConflict{DoNothing: true}).Create(&stylometry_samples).Error; stylometry_err != nil {
		return stylometry_err
	}
	if stylometry_err := stylometryRepairLegacySeedSamples(stylometry_db, stylometry_samples); stylometry_err != nil {
		return stylometry_err
	}

	stylometry_likes := stylometrySeedLikes(stylometry_now)
	if stylometry_err := stylometry_db.Clauses(clause.OnConflict{DoNothing: true}).Create(&stylometry_likes).Error; stylometry_err != nil {
		return stylometry_err
	}

	if stylometry_err := stylometryResetSequence(stylometry_db, "stylometry_users"); stylometry_err != nil {
		return stylometry_err
	}
	if stylometry_err := stylometryResetSequence(stylometry_db, "stylometry_text_samples"); stylometry_err != nil {
		return stylometry_err
	}
	return stylometryResetSequence(stylometry_db, "stylometry_likes")
}

// stylometryRepairLegacySeedSamples обновляет только старые строки, которые появились после укороченного seed.
func stylometryRepairLegacySeedSamples(stylometry_db *gorm.DB, stylometry_samples []model.StylometrySampleTextSample) error {
	stylometry_legacy_titles := map[uint][]string{
		1: {"Эталон Толстого"},
		2: {"Черновик Чехова"},
		3: {"Удалённый эталон Пушкина"},
	}
	stylometry_legacy_descriptions := map[uint][]string{
		1: {"Фрагмент из «Войны и мира»: длинные сложноподчинённые предложения, высокая частота союзов «и», «который»."},
		2: {"Черновик эталона: короткие предложения, минимум придаточных конструкций."},
		3: {"Насыщенный вводными конструкциями и повторами отрывок из «Преступления и наказания»."},
		4: {"Описательный фрагмент с плавным ритмом и умеренной частотой союзов."},
		5: {"Удалённая запись — используется для проверки того, что deleted не отображается в интерфейсе."},
	}

	for _, stylometry_sample := range stylometry_samples {
		stylometry_titles := stylometry_legacy_titles[stylometry_sample.StylometrySampleID]
		stylometry_descriptions := stylometry_legacy_descriptions[stylometry_sample.StylometrySampleID]
		if len(stylometry_titles) == 0 && len(stylometry_descriptions) == 0 {
			continue
		}

		stylometry_updates := map[string]interface{}{
			"title":                 stylometry_sample.StylometrySampleTitle,
			"short_description":     stylometry_sample.StylometrySampleDescr,
			"status":                stylometry_sample.StylometrySampleStatus,
			"image_url":             stylometry_sample.StylometrySampleImageURL,
			"video_url":             stylometry_sample.StylometrySampleVideoURL,
			"author_name":           stylometry_sample.StylometrySampleAuthorName,
			"avg_sentence_length":   stylometry_sample.StylometrySampleAvgSentLen,
			"conjunction_frequency": stylometry_sample.StylometrySampleConjFreq,
			"created_at":            stylometry_sample.StylometrySampleCreatedAt,
			"formed_at":             stylometry_sample.StylometrySampleFormedAt,
			"creator_id":            stylometry_sample.StylometrySampleCreatorID,
		}

		stylometry_query := stylometry_db.Model(&model.StylometrySampleTextSample{}).Where("id = ?", stylometry_sample.StylometrySampleID)
		if len(stylometry_titles) > 0 && len(stylometry_descriptions) > 0 {
			stylometry_query = stylometry_query.Where("title IN ? OR short_description IN ?", stylometry_titles, stylometry_descriptions)
		} else if len(stylometry_titles) > 0 {
			stylometry_query = stylometry_query.Where("title IN ?", stylometry_titles)
		} else {
			stylometry_query = stylometry_query.Where("short_description IN ?", stylometry_descriptions)
		}

		stylometry_err := stylometry_query.Updates(stylometry_updates).Error
		if stylometry_err != nil {
			return stylometry_err
		}
	}

	return nil
}

// stylometrySeedUsers собирает пользователей, чтобы все стартовые лайки имели существующие внешние ключи.
func stylometrySeedUsers(stylometry_now time.Time) []model.StylometrySampleUser {
	stylometry_users := make([]model.StylometrySampleUser, 0, 61)

	stylometry_users = append(stylometry_users, model.StylometrySampleUser{StylometrySampleID: 1, StylometrySampleName: "Демонстратор", StylometrySampleEmail: "demo@stylometry.local", StylometrySampleCreatedAt: stylometry_now})
	stylometry_users = append(stylometry_users, model.StylometrySampleUser{StylometrySampleID: 2, StylometrySampleName: "Читатель", StylometrySampleEmail: "reader@stylometry.local", StylometrySampleCreatedAt: stylometry_now})
	stylometry_users = append(stylometry_users, model.StylometrySampleUser{StylometrySampleID: 3, StylometrySampleName: "Редактор", StylometrySampleEmail: "editor@stylometry.local", StylometrySampleCreatedAt: stylometry_now})

	for stylometry_user_id := uint(4); stylometry_user_id <= 61; stylometry_user_id++ {
		stylometry_users = append(stylometry_users, model.StylometrySampleUser{
			StylometrySampleID:        stylometry_user_id,
			StylometrySampleName:      fmt.Sprintf("Пользователь %d", stylometry_user_id),
			StylometrySampleEmail:     fmt.Sprintf("user%d@stylometry.local", stylometry_user_id),
			StylometrySampleCreatedAt: stylometry_now,
		})
	}

	return stylometry_users
}

// stylometrySeedSamples собирает эталоны авторов и подключает к ним MinIO-медиа.
func stylometrySeedSamples(stylometry_now time.Time) []model.StylometrySampleTextSample {
	stylometry_formed_tolstoy := stylometry_now.Add(-119 * time.Hour)
	stylometry_formed_dostoevsky := stylometry_now.Add(-71 * time.Hour)
	stylometry_formed_turgenev := stylometry_now.Add(-47 * time.Hour)
	stylometry_formed_pushkin := stylometry_now.Add(-23 * time.Hour)
	stylometry_tolstoy, stylometry_tolstoy_metrics := stylometrySeedFragment("tolstoy")
	stylometry_chekhov, stylometry_chekhov_metrics := stylometrySeedFragment("chekhov")
	stylometry_dostoevsky, stylometry_dostoevsky_metrics := stylometrySeedFragment("dostoevsky")
	stylometry_turgenev, stylometry_turgenev_metrics := stylometrySeedFragment("turgenev")
	stylometry_pushkin, stylometry_pushkin_metrics := stylometrySeedFragment("pushkin")

	return []model.StylometrySampleTextSample{
		{
			StylometrySampleID:         1,
			StylometrySampleTitle:      "Эталон №1",
			StylometrySampleDescr:      stylometry_tolstoy.StylometrySampleText,
			StylometrySampleStatus:     model.StatusPublished,
			StylometrySampleImageURL:   stylometryMinIOMediaURL("images/tolstoy.webp"),
			StylometrySampleVideoURL:   stylometryMinIOMediaURL("videos/tolstoy.mp4"),
			StylometrySampleAuthorName: stylometry_tolstoy.StylometrySampleAuthorName,
			StylometrySampleAvgSentLen: stylometry_tolstoy_metrics.StylometrySampleAvgSentLen,
			StylometrySampleConjFreq:   stylometry_tolstoy_metrics.StylometrySampleConjFreq,
			StylometrySampleCreatedAt:  stylometry_now.Add(-120 * time.Hour),
			StylometrySampleFormedAt:   &stylometry_formed_tolstoy,
			StylometrySampleCreatorID:  1,
		},
		{
			StylometrySampleID:         2,
			StylometrySampleTitle:      "Эталон №2",
			StylometrySampleDescr:      stylometry_chekhov.StylometrySampleText,
			StylometrySampleStatus:     model.StatusDraft,
			StylometrySampleImageURL:   stylometryMinIOMediaURL("images/chekhov.webp"),
			StylometrySampleVideoURL:   stylometryMinIOMediaURL("videos/chekhov.mp4"),
			StylometrySampleAuthorName: stylometry_chekhov.StylometrySampleAuthorName,
			StylometrySampleAvgSentLen: stylometry_chekhov_metrics.StylometrySampleAvgSentLen,
			StylometrySampleConjFreq:   stylometry_chekhov_metrics.StylometrySampleConjFreq,
			StylometrySampleCreatedAt:  stylometry_now.Add(-96 * time.Hour),
			StylometrySampleFormedAt:   nil,
			StylometrySampleCreatorID:  1,
		},
		{
			StylometrySampleID:         3,
			StylometrySampleTitle:      "Эталон №3",
			StylometrySampleDescr:      stylometry_dostoevsky.StylometrySampleText,
			StylometrySampleStatus:     model.StatusPublished,
			StylometrySampleImageURL:   stylometryMinIOMediaURL("images/dostoevsky.webp"),
			StylometrySampleVideoURL:   stylometryMinIOMediaURL("videos/dostoevsky.mp4"),
			StylometrySampleAuthorName: stylometry_dostoevsky.StylometrySampleAuthorName,
			StylometrySampleAvgSentLen: stylometry_dostoevsky_metrics.StylometrySampleAvgSentLen,
			StylometrySampleConjFreq:   stylometry_dostoevsky_metrics.StylometrySampleConjFreq,
			StylometrySampleCreatedAt:  stylometry_now.Add(-72 * time.Hour),
			StylometrySampleFormedAt:   &stylometry_formed_dostoevsky,
			StylometrySampleCreatorID:  1,
		},
		{
			StylometrySampleID:         4,
			StylometrySampleTitle:      "Эталон №4",
			StylometrySampleDescr:      stylometry_turgenev.StylometrySampleText,
			StylometrySampleStatus:     model.StatusPublished,
			StylometrySampleImageURL:   stylometryMinIOMediaURL("images/turgenev.webp"),
			StylometrySampleVideoURL:   stylometryMinIOMediaURL("videos/turgenev.mp4"),
			StylometrySampleAuthorName: stylometry_turgenev.StylometrySampleAuthorName,
			StylometrySampleAvgSentLen: stylometry_turgenev_metrics.StylometrySampleAvgSentLen,
			StylometrySampleConjFreq:   stylometry_turgenev_metrics.StylometrySampleConjFreq,
			StylometrySampleCreatedAt:  stylometry_now.Add(-48 * time.Hour),
			StylometrySampleFormedAt:   &stylometry_formed_turgenev,
			StylometrySampleCreatorID:  1,
		},
		{
			StylometrySampleID:         5,
			StylometrySampleTitle:      "Эталон №5 (удалён)",
			StylometrySampleDescr:      stylometry_pushkin.StylometrySampleText,
			StylometrySampleStatus:     model.StatusDeleted,
			StylometrySampleImageURL:   stylometryMinIOMediaURL("images/pushkin.webp"),
			StylometrySampleVideoURL:   stylometryMinIOMediaURL("videos/pushkin.mp4"),
			StylometrySampleAuthorName: stylometry_pushkin.StylometrySampleAuthorName,
			StylometrySampleAvgSentLen: stylometry_pushkin_metrics.StylometrySampleAvgSentLen,
			StylometrySampleConjFreq:   stylometry_pushkin_metrics.StylometrySampleConjFreq,
			StylometrySampleCreatedAt:  stylometry_now.Add(-24 * time.Hour),
			StylometrySampleFormedAt:   &stylometry_formed_pushkin,
			StylometrySampleCreatorID:  1,
		},
	}
}

// stylometrySeedFragment возвращает сохранённый фрагмент и рассчитанные параметры.
func stylometrySeedFragment(stylometry_key string) (analysis.StylometrySampleFragment, analysis.StylometrySampleMetrics) {
	stylometry_fragment, stylometry_ok := analysis.FragmentByKey(stylometry_key)
	if !stylometry_ok {
		stylometry_fragment = analysis.ResolveFragment("", "", "")
	}
	stylometry_metrics := analysis.AnalyzeFragment(stylometry_fragment.StylometrySampleText)
	return stylometry_fragment, stylometry_metrics
}

// stylometrySeedLikes собирает стартовые лайки для таблицы многие-ко-многим.
func stylometrySeedLikes(stylometry_now time.Time) []model.StylometrySampleLike {
	stylometry_likes_by_sample := map[uint][]uint{
		1: []uint{3, 7, 8, 12, 15, 21, 24, 30, 31, 42, 55, 61},
		2: []uint{5, 9, 14},
		3: []uint{2, 6, 11, 18, 23, 29, 37},
		4: []uint{4, 10, 16, 25, 33},
		5: []uint{1, 19},
	}

	stylometry_likes := make([]model.StylometrySampleLike, 0, 29)
	stylometry_like_id := uint(1)

	for stylometry_sample_id := uint(1); stylometry_sample_id <= 5; stylometry_sample_id++ {
		for _, stylometry_user_id := range stylometry_likes_by_sample[stylometry_sample_id] {
			stylometry_likes = append(stylometry_likes, model.StylometrySampleLike{
				StylometrySampleLikeID:       stylometry_like_id,
				StylometrySampleUserID:       stylometry_user_id,
				StylometrySampleTextSampleID: stylometry_sample_id,
				StylometrySampleCreatedAt:    stylometry_now.Add(-time.Duration(stylometry_like_id) * time.Hour),
			})
			stylometry_like_id++
		}
	}

	return stylometry_likes
}

// stylometryMinIOMediaURL строит публичную ссылку на файл в MinIO.
func stylometryMinIOMediaURL(stylometry_object_key string) string {
	stylometry_public_url := stylometryEnv("MINIO_PUBLIC_URL", "http://127.0.0.1:9000")
	stylometry_bucket := stylometryEnv("MINIO_BUCKET", "stylometry")
	return fmt.Sprintf("%s/%s/%s", stylometry_public_url, stylometry_bucket, stylometry_object_key)
}

// stylometryResetSequence сдвигает PostgreSQL sequence после seed-строк с явными id.
func stylometryResetSequence(stylometry_db *gorm.DB, stylometry_table string) error {
	stylometry_sql := fmt.Sprintf("SELECT setval(pg_get_serial_sequence('%s', 'id'), COALESCE((SELECT MAX(id) FROM %s), 1))", stylometry_table, stylometry_table)
	return stylometry_db.Exec(stylometry_sql).Error
}
