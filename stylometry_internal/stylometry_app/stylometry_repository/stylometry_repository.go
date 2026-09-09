package repository

import (
	"fmt"
	"os"
)

// StylometryStatus — статус образца в коллекции эталонов.
type StylometryStatus string

const (
	StylometryStatusDraft     StylometryStatus = "draft"
	StylometryStatusPublished StylometryStatus = "published"
	StylometryStatusDeleted   StylometryStatus = "deleted"
)

// StylometrySample — эталонный образец текста известного автора
// с вычисленными стилометрическими маркерами.
type StylometrySample struct {
	StylometryID         int
	StylometryTitle      string
	StylometryAuthorName string
	StylometryAvgSentLen float64
	StylometryConjFreq   float64
	StylometryStatus     StylometryStatus
	StylometryDescr      string
	StylometryImageURL   string
	StylometryVideoURL   string
	StylometryLikes      []int
}

func (stylometry_sample StylometrySample) LikesCount() int {
	return len(stylometry_sample.StylometryLikes)
}

func (stylometry_sample StylometrySample) IsVisible() bool {
	return stylometry_sample.StylometryStatus != StylometryStatusDeleted
}

// StylometryRepository — коллекция образцов (без БД, как в задании лаб. работы №1).
type StylometryRepository struct {
	stylometry_samples []StylometrySample
}

func NewStylometryRepository() (*StylometryRepository, error) {
	stylometry_public_url := stylometry_get_env("MINIO_PUBLIC_URL", "http://127.0.0.1:9000")
	stylometry_bucket := stylometry_get_env("MINIO_BUCKET", "stylometry")

	return &StylometryRepository{
		stylometry_samples: stylometry_create_samples(stylometry_public_url, stylometry_bucket),
	}, nil
}

func stylometry_get_env(stylometry_key, stylometry_fallback string) string {
	if stylometry_value := os.Getenv(stylometry_key); stylometry_value != "" {
		return stylometry_value
	}
	return stylometry_fallback
}

func stylometry_media_url(stylometry_public_url, stylometry_bucket, stylometry_object_key string) string {
	return fmt.Sprintf("%s/%s/%s", stylometry_public_url, stylometry_bucket, stylometry_object_key)
}

func stylometry_like_ids(stylometry_count int) []int {
	stylometry_ids := make([]int, stylometry_count)
	for stylometry_i := 0; stylometry_i < stylometry_count; stylometry_i++ {
		stylometry_ids[stylometry_i] = stylometry_i + 1
	}
	return stylometry_ids
}

func stylometry_create_samples(stylometry_public_url, stylometry_bucket string) []StylometrySample {
	return []StylometrySample{
		{StylometryID: 1, StylometryTitle: "Эталон №1", StylometryAuthorName: "Л. Н. Толстой", StylometryAvgSentLen: 24.6, StylometryConjFreq: 8.4, StylometryStatus: StylometryStatusPublished, StylometryDescr: "Фрагмент из «Войны и мира»: длинные сложноподчинённые предложения, высокая частота союзов «и», «который».", StylometryImageURL: stylometry_media_url(stylometry_public_url, stylometry_bucket, "images/tolstoy.webp"), StylometryVideoURL: stylometry_media_url(stylometry_public_url, stylometry_bucket, "videos/tolstoy.mp4"), StylometryLikes: stylometry_like_ids(120)},
		{StylometryID: 2, StylometryTitle: "Эталон №2", StylometryAuthorName: "А. П. Чехов", StylometryAvgSentLen: 11.2, StylometryConjFreq: 5.1, StylometryStatus: StylometryStatusDraft, StylometryDescr: "Черновик эталона: короткие предложения, минимум придаточных конструкций.", StylometryImageURL: stylometry_media_url(stylometry_public_url, stylometry_bucket, "images/chekhov.webp"), StylometryVideoURL: stylometry_media_url(stylometry_public_url, stylometry_bucket, "videos/chekhov.mp4"), StylometryLikes: stylometry_like_ids(14)},
		{StylometryID: 3, StylometryTitle: "Эталон №3", StylometryAuthorName: "Ф. М. Достоевский", StylometryAvgSentLen: 19.8, StylometryConjFreq: 7.2, StylometryStatus: StylometryStatusPublished, StylometryDescr: "Насыщенный вводными конструкциями и повторами отрывок из «Преступления и наказания».", StylometryImageURL: stylometry_media_url(stylometry_public_url, stylometry_bucket, "images/dostoevsky.webp"), StylometryVideoURL: stylometry_media_url(stylometry_public_url, stylometry_bucket, "videos/dostoevsky.mp4"), StylometryLikes: stylometry_like_ids(56)},
		{StylometryID: 4, StylometryTitle: "Эталон №4", StylometryAuthorName: "И. С. Тургенев", StylometryAvgSentLen: 16.3, StylometryConjFreq: 6.0, StylometryStatus: StylometryStatusPublished, StylometryDescr: "Описательный фрагмент с плавным ритмом и умеренной частотой союзов.", StylometryImageURL: stylometry_media_url(stylometry_public_url, stylometry_bucket, "images/turgenev.webp"), StylometryVideoURL: stylometry_media_url(stylometry_public_url, stylometry_bucket, "videos/turgenev.mp4"), StylometryLikes: stylometry_like_ids(41)},
		{StylometryID: 5, StylometryTitle: "Эталон №5 (удалён)", StylometryAuthorName: "А. С. Пушкин", StylometryAvgSentLen: 14.0, StylometryConjFreq: 5.8, StylometryStatus: StylometryStatusDeleted, StylometryDescr: "Удалённая запись — используется для проверки того, что deleted не отображается в интерфейсе.", StylometryImageURL: stylometry_media_url(stylometry_public_url, stylometry_bucket, "images/pushkin.webp"), StylometryVideoURL: stylometry_media_url(stylometry_public_url, stylometry_bucket, "videos/pushkin.mp4"), StylometryLikes: stylometry_like_ids(9)},
	}
}

func (stylometry_repository *StylometryRepository) VisibleSamples() []StylometrySample {
	stylometry_result := make([]StylometrySample, 0, len(stylometry_repository.stylometry_samples))
	for _, stylometry_sample := range stylometry_repository.stylometry_samples {
		if stylometry_sample.IsVisible() {
			stylometry_result = append(stylometry_result, stylometry_sample)
		}
	}
	return stylometry_result
}

func (stylometry_repository *StylometryRepository) SampleByID(stylometry_id int) (StylometrySample, bool) {
	for _, stylometry_sample := range stylometry_repository.VisibleSamples() {
		if stylometry_sample.StylometryID == stylometry_id {
			return stylometry_sample, true
		}
	}
	return StylometrySample{}, false
}

func (stylometry_repository *StylometryRepository) DraftSample() (StylometrySample, bool) {
	for _, stylometry_sample := range stylometry_repository.stylometry_samples {
		if stylometry_sample.StylometryStatus == StylometryStatusDraft {
			return stylometry_sample, true
		}
	}
	return StylometrySample{}, false
}

func (stylometry_repository *StylometryRepository) FilteredSamples(stylometry_max_avg_len *float64) []StylometrySample {
	stylometry_result := make([]StylometrySample, 0)
	for _, stylometry_sample := range stylometry_repository.VisibleSamples() {
		if stylometry_max_avg_len != nil && stylometry_sample.StylometryAvgSentLen > *stylometry_max_avg_len {
			continue
		}
		stylometry_result = append(stylometry_result, stylometry_sample)
	}
	return stylometry_result
}
