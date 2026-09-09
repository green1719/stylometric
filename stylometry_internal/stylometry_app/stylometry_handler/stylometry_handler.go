package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"stylometry/stylometry_internal/stylometry_app/stylometry_repository"
)

type StylometryHandler struct {
	Repository *repository.StylometryRepository
}

func NewStylometryHandler(stylometry_r *repository.StylometryRepository) *StylometryHandler {
	return &StylometryHandler{Repository: stylometry_r}
}

// GET /samples — лента, для панели навигации (без id, показывает первый доступный образец)
func (stylometry_h *StylometryHandler) GetStylometryFeed(stylometry_ctx *gin.Context) {
	stylometry_samples := stylometry_h.Repository.VisibleSamples()
	if len(stylometry_samples) == 0 {
		stylometry_ctx.String(http.StatusNotFound, "нет доступных эталонных образцов")
		return
	}

	stylometry_index := 0
	if stylometry_ctx.Query("next") == "true" {
		stylometry_index = 1 % len(stylometry_samples)
	}

	stylometry_ctx.HTML(http.StatusOK, "stylometry_feed.html", gin.H{"stylometry_sample": stylometry_samples[stylometry_index]})
}

// GET /samples/:id?next=true
func (stylometry_h *StylometryHandler) GetStylometryFeedByID(stylometry_ctx *gin.Context) {
	stylometry_samples := stylometry_h.Repository.VisibleSamples()
	if len(stylometry_samples) == 0 {
		stylometry_ctx.String(http.StatusNotFound, "нет доступных эталонных образцов")
		return
	}

	stylometry_id, stylometry_err := strconv.Atoi(stylometry_ctx.Param("id"))
	if stylometry_err != nil {
		logrus.Error(stylometry_err)
		stylometry_ctx.String(http.StatusBadRequest, "некорректный id")
		return
	}

	stylometry_index := -1
	for stylometry_i, stylometry_sample := range stylometry_samples {
		if stylometry_sample.StylometryID == stylometry_id {
			stylometry_index = stylometry_i
			break
		}
	}
	if stylometry_index == -1 {
		stylometry_ctx.String(http.StatusNotFound, "образец с таким id отсутствует или удалён")
		return
	}

	if stylometry_ctx.Query("next") == "true" {
		stylometry_index = (stylometry_index + 1) % len(stylometry_samples)
	}

	stylometry_ctx.HTML(http.StatusOK, "stylometry_feed.html", gin.H{"stylometry_sample": stylometry_samples[stylometry_index]})
}

// GET /samples/draft — страница "Добавление"
func (stylometry_h *StylometryHandler) GetStylometryDraft(stylometry_ctx *gin.Context) {
	stylometry_draft, stylometry_ok := stylometry_h.Repository.DraftSample()
	if !stylometry_ok {
		stylometry_ctx.String(http.StatusNotFound, "черновик эталона не найден")
		return
	}
	stylometry_ctx.HTML(http.StatusOK, "stylometry_draft.html", gin.H{"stylometry_sample": stylometry_draft})
}

// GET /samples/grid?stylometry_avg_len=... — страница "Плитка"
func (stylometry_h *StylometryHandler) GetStylometryGrid(stylometry_ctx *gin.Context) {
	stylometry_raw_filter := strings.TrimSpace(stylometry_ctx.Query("stylometry_avg_len"))

	var stylometry_max_avg_len *float64
	if stylometry_raw_filter != "" {
		stylometry_normalized := strings.ReplaceAll(stylometry_raw_filter, ",", ".")
		stylometry_num, stylometry_err := strconv.ParseFloat(stylometry_normalized, 64)
		if stylometry_err != nil || stylometry_num < 0 {
			stylometry_ctx.String(http.StatusBadRequest, "stylometry_avg_len должен быть неотрицательным числом, например 18.4")
			return
		}
		stylometry_max_avg_len = &stylometry_num
	}

	stylometry_samples := stylometry_h.Repository.FilteredSamples(stylometry_max_avg_len)

	stylometry_cover_url := ""
	if len(stylometry_samples) > 0 {
		stylometry_cover_url = stylometry_samples[0].StylometryImageURL
	}

	stylometry_ctx.HTML(http.StatusOK, "stylometry_grid.html", gin.H{
		"has_stylometry_samples": len(stylometry_samples) > 0,
		"stylometry_samples":     stylometry_samples,
		"stylometry_avg_len":     stylometry_raw_filter,
		"stylometry_filter_on":   stylometry_max_avg_len != nil,
		"stylometry_cover_url":   stylometry_cover_url,
	})
}
