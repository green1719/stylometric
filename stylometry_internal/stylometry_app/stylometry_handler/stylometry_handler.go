package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"stylometry/stylometry_internal/stylometry_app/stylometry_analysis"
	"stylometry/stylometry_internal/stylometry_app/stylometry_model"
	"stylometry/stylometry_internal/stylometry_app/stylometry_repository"
)

// Границы ползунка фильтрации по средней длине предложения.
const (
	stylometryAvgLenMin  = 0.0
	stylometryAvgLenMax  = 30.0
	stylometryAvgLenStep = 0.5
)

// StylometrySampleHandler хранит зависимости HTTP-слоя.
type StylometrySampleHandler struct {
	Repository *repository.StylometrySampleRepository
}

// NewStylometrySampleHandler создаёт HTTP-контроллер.
func NewStylometrySampleHandler(stylometry_r *repository.StylometrySampleRepository) *StylometrySampleHandler {
	return &StylometrySampleHandler{Repository: stylometry_r}
}

// stylometryFormatLen красиво форматирует float64 для HTML.
func stylometryFormatLen(stylometry_value float64) string {
	return strconv.FormatFloat(stylometry_value, 'f', -1, 64)
}

// stylometryParseFloat читает число из формы.
func stylometryParseFloat(stylometry_raw string) (float64, error) {
	stylometry_normalized := strings.ReplaceAll(strings.TrimSpace(stylometry_raw), ",", ".")
	return strconv.ParseFloat(stylometry_normalized, 64)
}

// GET /samples — лента; при stylometry_id показывает конкретную опубликованную карточку.
func (stylometry_h *StylometrySampleHandler) GetStylometrySampleFeed(stylometry_ctx *gin.Context) {
	stylometry_raw_id := strings.TrimSpace(stylometry_ctx.Query("stylometry_id"))
	if stylometry_raw_id != "" {
		stylometry_id, stylometry_err := strconv.ParseUint(stylometry_raw_id, 10, 64)
		if stylometry_err != nil {
			stylometry_ctx.String(http.StatusBadRequest, "некорректный id")
			return
		}

		stylometry_sample, stylometry_ok, stylometry_err := stylometry_h.Repository.SampleByID(uint(stylometry_id))
		if stylometry_err != nil {
			logrus.Error(stylometry_err)
			stylometry_ctx.String(http.StatusInternalServerError, "ошибка базы данных")
			return
		}
		if !stylometry_ok {
			stylometry_ctx.String(http.StatusNotFound, "образец с таким id отсутствует, не опубликован или удалён")
			return
		}

		stylometry_ctx.HTML(http.StatusOK, "stylometry_feed.html", gin.H{"stylometry_sample": stylometry_sample})
		return
	}

	stylometry_samples, stylometry_err := stylometry_h.Repository.PublishedSamples()
	if stylometry_err != nil {
		logrus.Error(stylometry_err)
		stylometry_ctx.String(http.StatusInternalServerError, "ошибка базы данных")
		return
	}
	if len(stylometry_samples) == 0 {
		stylometry_ctx.String(http.StatusNotFound, "нет опубликованных эталонных образцов")
		return
	}

	stylometry_index := 0
	stylometry_ctx.HTML(http.StatusOK, "stylometry_feed.html", gin.H{"stylometry_sample": stylometry_samples[stylometry_index]})
}

// GET /samples/draft — страница добавления или публикации черновика.
func (stylometry_h *StylometrySampleHandler) GetStylometrySampleDraft(stylometry_ctx *gin.Context) {
	stylometry_draft, stylometry_has_draft, stylometry_err := stylometry_h.Repository.DraftSample()
	if stylometry_err != nil {
		logrus.Error(stylometry_err)
		stylometry_ctx.String(http.StatusInternalServerError, "ошибка базы данных")
		return
	}

	if !stylometry_has_draft {
		stylometry_draft = model.StylometrySampleTextSample{
			StylometrySampleTitle:      "",
			StylometrySampleImageURL:   repository.DefaultImageURL,
			StylometrySampleVideoURL:   repository.DefaultVideoURL,
			StylometrySampleStatus:     model.StatusDraft,
			StylometrySampleDescr:      "",
			StylometrySampleAuthorName: "",
		}
	}

	stylometry_ctx.HTML(http.StatusOK, "stylometry_draft.html", gin.H{
		"stylometry_sample":    stylometry_draft,
		"stylometry_has_draft": stylometry_has_draft,
		"stylometry_fragments": analysis.ClassicFragments(),
	})
}

// POST /samples/draft — создаёт черновик через ORM.
func (stylometry_h *StylometrySampleHandler) PostStylometrySampleCreateDraft(stylometry_ctx *gin.Context) {
	stylometry_title := strings.TrimSpace(stylometry_ctx.PostForm("stylometry_title"))
	if stylometry_title == "" {
		stylometry_title = "Новый эталон"
	}

	stylometry_fragment_key := strings.TrimSpace(stylometry_ctx.PostForm("stylometry_fragment_key"))
	stylometry_fragment_text := strings.TrimSpace(stylometry_ctx.PostForm("stylometry_fragment"))

	_, stylometry_err := stylometry_h.Repository.CreateDraft(stylometry_title, stylometry_fragment_key, stylometry_fragment_text)
	if stylometry_err != nil {
		logrus.Error(stylometry_err)
		stylometry_ctx.String(http.StatusInternalServerError, "не удалось создать черновик")
		return
	}

	stylometry_ctx.Redirect(http.StatusSeeOther, "/samples/draft")
}

// POST /samples/draft/publish — публикует черновик через ORM.
func (stylometry_h *StylometrySampleHandler) PostStylometrySamplePublishDraft(stylometry_ctx *gin.Context) {
	stylometry_description := strings.TrimSpace(stylometry_ctx.PostForm("stylometry_descr"))
	stylometry_author := strings.TrimSpace(stylometry_ctx.PostForm("stylometry_author_name"))
	if stylometry_author == "" {
		stylometry_author = "Не указан"
	}

	stylometry_avg_len, stylometry_err := stylometryParseFloat(stylometry_ctx.PostForm("stylometry_avg_sent_len"))
	if stylometry_err != nil {
		stylometry_ctx.String(http.StatusBadRequest, "средняя длина предложения должна быть числом")
		return
	}

	stylometry_conj_freq, stylometry_err := stylometryParseFloat(stylometry_ctx.PostForm("stylometry_conj_freq"))
	if stylometry_err != nil {
		stylometry_ctx.String(http.StatusBadRequest, "частота союзов должна быть числом")
		return
	}

	stylometry_err = stylometry_h.Repository.PublishDraft(stylometry_description, stylometry_author, stylometry_avg_len, stylometry_conj_freq)
	if stylometry_err != nil {
		logrus.Error(stylometry_err)
		stylometry_ctx.String(http.StatusInternalServerError, "не удалось опубликовать черновик")
		return
	}

	stylometry_ctx.Redirect(http.StatusSeeOther, "/samples/grid")
}

// POST /samples/delete — логически удаляет карточку через SQL UPDATE.
func (stylometry_h *StylometrySampleHandler) PostStylometrySampleDelete(stylometry_ctx *gin.Context) {
	stylometry_raw_id := strings.TrimSpace(stylometry_ctx.PostForm("stylometry_id"))
	stylometry_id, stylometry_err := strconv.ParseUint(stylometry_raw_id, 10, 64)
	if stylometry_err != nil {
		stylometry_ctx.String(http.StatusBadRequest, "некорректный id")
		return
	}

	stylometry_err = stylometry_h.Repository.DeleteSampleSQL(uint(stylometry_id))
	if stylometry_err != nil {
		logrus.Error(stylometry_err)
		stylometry_ctx.String(http.StatusInternalServerError, "не удалось удалить карточку")
		return
	}

	stylometry_ctx.Redirect(http.StatusSeeOther, "/samples/grid")
}

// GET /samples/grid — плитка с поиском по названию и фильтром по средней длине.
func (stylometry_h *StylometrySampleHandler) GetStylometrySampleGrid(stylometry_ctx *gin.Context) {
	stylometry_search := strings.TrimSpace(stylometry_ctx.Query("stylometry_search"))
	stylometry_raw_filter := strings.TrimSpace(stylometry_ctx.Query("stylometry_avg_len"))

	var stylometry_max_avg_len *float64
	if stylometry_raw_filter != "" {
		stylometry_num, stylometry_err := stylometryParseFloat(stylometry_raw_filter)
		if stylometry_err != nil || stylometry_num < stylometryAvgLenMin || stylometry_num > stylometryAvgLenMax {
			stylometry_ctx.String(http.StatusBadRequest, "stylometry_avg_len должен быть числом от %s до %s", stylometryFormatLen(stylometryAvgLenMin), stylometryFormatLen(stylometryAvgLenMax))
			return
		}
		stylometry_max_avg_len = &stylometry_num
	}

	stylometry_samples, stylometry_err := stylometry_h.Repository.FilteredSamples(stylometry_search, stylometry_max_avg_len)
	if stylometry_err != nil {
		logrus.Error(stylometry_err)
		stylometry_ctx.String(http.StatusInternalServerError, "ошибка базы данных")
		return
	}

	stylometry_cover_url := repository.DefaultImageURL
	if len(stylometry_samples) > 0 {
		stylometry_cover_url = stylometry_samples[0].StylometrySampleImageURL
	}

	stylometry_slider_value := stylometryFormatLen(stylometryAvgLenMax)
	if stylometry_max_avg_len != nil {
		stylometry_slider_value = stylometryFormatLen(*stylometry_max_avg_len)
	}

	stylometry_ctx.HTML(http.StatusOK, "stylometry_grid.html", gin.H{
		"has_stylometry_samples":   len(stylometry_samples) > 0,
		"stylometry_samples":       stylometry_samples,
		"stylometry_search":        stylometry_search,
		"stylometry_avg_len":       stylometry_raw_filter,
		"stylometry_filter_on":     stylometry_max_avg_len != nil,
		"stylometry_cover_url":     stylometry_cover_url,
		"stylometry_avg_len_value": stylometry_slider_value,
		"stylometry_avg_len_min":   stylometryFormatLen(stylometryAvgLenMin),
		"stylometry_avg_len_max":   stylometryFormatLen(stylometryAvgLenMax),
		"stylometry_avg_len_step":  stylometryFormatLen(stylometryAvgLenStep),
	})
}
