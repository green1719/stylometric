package handler

import (
	"errors"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"stylometry/stylometry_internal/stylometry_app/stylometry_analysis"
	"stylometry/stylometry_internal/stylometry_app/stylometry_media"
	"stylometry/stylometry_internal/stylometry_app/stylometry_serializer"
	"stylometry/stylometry_internal/stylometry_app/stylometry_session"
)

func stylometryAPIUserID() uint {
	return stylometry_session.CurrentUser().ID
}

func stylometryAPIFormFile(stylometry_ctx *gin.Context, stylometry_names ...string) (*multipart.FileHeader, error) {
	for _, stylometry_name := range stylometry_names {
		stylometry_file, stylometry_err := stylometry_ctx.FormFile(stylometry_name)
		if stylometry_err == nil {
			return stylometry_file, nil
		}
		if !errors.Is(stylometry_err, http.ErrMissingFile) {
			return nil, stylometry_err
		}
	}
	return nil, http.ErrMissingFile
}

type stylometryAPILikeRequest struct {
	StylometryLike *int `form:"stylometry_like" json:"stylometry_like"`
	Like           *int `form:"like" json:"like"`
}

type stylometryAPIUserRequest struct {
	Name     string `form:"stylometry_sample_name" json:"stylometry_sample_name"`
	Password string `form:"stylometry_sample_password" json:"stylometry_sample_password"`
}

// stylometrySplitDraftText separates the author from the text entered in one field.
func stylometrySplitDraftText(stylometry_text string) (string, string, error) {
	stylometry_text = strings.ReplaceAll(stylometry_text, "\r\n", "\n")
	stylometry_text = strings.ReplaceAll(stylometry_text, "\r", "\n")
	stylometry_parts := strings.SplitN(stylometry_text, "\n", 2)
	if len(stylometry_parts) != 2 {
		return "", "", errors.New("поле text должно содержать автора в первой строке и текстовый отрывок ниже")
	}

	stylometry_text_creator := strings.TrimSpace(stylometry_parts[0])
	stylometry_description := stylometry_parts[1]
	if stylometry_text_creator == "" || strings.TrimSpace(stylometry_description) == "" {
		return "", "", errors.New("автор и текстовый отрывок в поле text не должны быть пустыми")
	}

	return stylometry_text_creator, stylometry_description, nil
}

// GET /api/samples
func (stylometry_h *SampleHandler) APIGetSampleList(stylometry_ctx *gin.Context) {
	stylometry_raw_filter := strings.TrimSpace(stylometry_ctx.Query("stylometry_avg_len"))

	var stylometry_max_avg_len *float64
	if stylometry_raw_filter != "" {
		stylometry_num, stylometry_err := stylometryParseFloat(stylometry_raw_filter)
		if stylometry_err != nil || stylometry_num < stylometryAvgLenMin || stylometry_num > stylometryAvgLenMax {
			stylometry_ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "stylometry_avg_len должен быть числом от 0 до 30",
			})
			return
		}
		stylometry_max_avg_len = &stylometry_num
	}

	stylometry_samples, stylometry_err := stylometry_h.Repository.FilteredSamples(stylometry_max_avg_len)
	if stylometry_err != nil {
		logrus.Error(stylometry_err)
		stylometry_ctx.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка базы данных"})
		return
	}

	stylometry_ctx.JSON(http.StatusOK, gin.H{
		"items": serializer.SampleListResponse(stylometry_samples, stylometryAPIUserID()),
	})
}

// GET /api/samples/feed
func (stylometry_h *SampleHandler) APIGetSampleFeed(stylometry_ctx *gin.Context) {
	stylometry_raw_id := strings.TrimSpace(stylometry_ctx.Query("stylometry_id"))
	stylometry_next := strings.EqualFold(strings.TrimSpace(stylometry_ctx.Query("next")), "true")

	var stylometry_sample_id *uint
	if stylometry_raw_id != "" {
		stylometry_id, stylometry_err := strconv.ParseUint(stylometry_raw_id, 10, 64)
		if stylometry_err != nil {
			stylometry_ctx.JSON(http.StatusBadRequest, gin.H{"error": "некорректный stylometry_id"})
			return
		}
		stylometry_uint_id := uint(stylometry_id)
		stylometry_sample_id = &stylometry_uint_id
	}

	stylometry_sample, stylometry_ok, stylometry_err := stylometry_h.Repository.FeedSample(stylometry_sample_id, stylometry_next)
	if stylometry_err != nil {
		logrus.Error(stylometry_err)
		stylometry_ctx.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка базы данных"})
		return
	}
	if !stylometry_ok {
		stylometry_ctx.JSON(http.StatusNotFound, gin.H{"error": "опубликованный образец не найден"})
		return
	}

	stylometry_ctx.JSON(http.StatusOK, serializer.SampleResponse(stylometry_sample, stylometryAPIUserID()))
}

// GET /api/samples/draft
func (stylometry_h *SampleHandler) APIGetSampleDraft(stylometry_ctx *gin.Context) {
	stylometry_draft, stylometry_has_draft, stylometry_err := stylometry_h.Repository.DraftSample()
	if stylometry_err != nil {
		logrus.Error(stylometry_err)
		stylometry_ctx.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка базы данных"})
		return
	}
	if !stylometry_has_draft {
		stylometry_ctx.JSON(http.StatusOK, gin.H{
			"has_draft": false,
			"draft":     nil,
		})
		return
	}

	stylometry_ctx.JSON(http.StatusOK, gin.H{
		"has_draft": true,
		"draft":     serializer.SampleResponse(stylometry_draft, stylometryAPIUserID()),
	})
}

// POST /api/samples
func (stylometry_h *SampleHandler) APIPostSampleDraft(stylometry_ctx *gin.Context) {
	stylometry_text_creator, stylometry_description, stylometry_err := stylometrySplitDraftText(stylometry_ctx.PostForm("text"))
	if stylometry_err != nil {
		stylometry_ctx.JSON(http.StatusBadRequest, gin.H{"error": stylometry_err.Error()})
		return
	}

	stylometry_image_file, stylometry_err := stylometryAPIFormFile(stylometry_ctx, "image")
	if errors.Is(stylometry_err, http.ErrMissingFile) {
		stylometry_ctx.JSON(http.StatusBadRequest, gin.H{"error": "нужно приложить файл изображения в поле image"})
		return
	}
	if stylometry_err != nil {
		logrus.Error(stylometry_err)
		stylometry_ctx.JSON(http.StatusBadRequest, gin.H{"error": "не удалось прочитать файл изображения"})
		return
	}

	stylometry_video_file, stylometry_err := stylometryAPIFormFile(stylometry_ctx, "video")
	if errors.Is(stylometry_err, http.ErrMissingFile) {
		stylometry_ctx.JSON(http.StatusBadRequest, gin.H{"error": "нужно приложить файл видео в поле video"})
		return
	}
	if stylometry_err != nil {
		logrus.Error(stylometry_err)
		stylometry_ctx.JSON(http.StatusBadRequest, gin.H{"error": "не удалось прочитать файл видео"})
		return
	}

	stylometry_image_url, stylometry_err := media.UploadFile(stylometry_ctx.Request.Context(), stylometry_image_file, "stylometry_images")
	if stylometry_err != nil {
		logrus.Error(stylometry_err)
		stylometry_ctx.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось загрузить изображение в MinIO"})
		return
	}

	stylometry_video_url, stylometry_err := media.UploadFile(stylometry_ctx.Request.Context(), stylometry_video_file, "stylometry_videos")
	if stylometry_err != nil {
		logrus.Error(stylometry_err)
		stylometry_ctx.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось загрузить видео в MinIO"})
		return
	}

	stylometry_sample, stylometry_err := stylometry_h.Repository.CreateDraft(
		stylometry_text_creator,
		stylometry_description,
		stylometry_image_url,
		stylometry_video_url,
	)
	if stylometry_err != nil {
		logrus.Error(stylometry_err)
		stylometry_ctx.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось сохранить черновик"})
		return
	}

	stylometry_ctx.JSON(http.StatusCreated, gin.H{
		"draft": serializer.SampleResponse(stylometry_sample, stylometryAPIUserID()),
	})
}

// PUT /api/samples/draft/publish
func (stylometry_h *SampleHandler) APIPutSamplePublishDraft(stylometry_ctx *gin.Context) {
	stylometry_draft, stylometry_has_draft, stylometry_err := stylometry_h.Repository.DraftSample()
	if stylometry_err != nil {
		logrus.Error(stylometry_err)
		stylometry_ctx.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка базы данных"})
		return
	}
	if !stylometry_has_draft {
		stylometry_ctx.JSON(http.StatusNotFound, gin.H{"error": "черновик текущего пользователя не найден"})
		return
	}

	stylometry_metrics := analysis.AnalyzeText(stylometry_draft.ShortDescription)
	stylometry_err = stylometry_h.Repository.PublishDraft(
		stylometry_draft.ShortDescription,
		stylometry_metrics.AverageSentenceLength,
		stylometry_metrics.ConjunctionFrequency,
	)
	if stylometry_err != nil {
		logrus.Error(stylometry_err)
		stylometry_ctx.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось опубликовать черновик"})
		return
	}

	stylometry_published, stylometry_ok, stylometry_err := stylometry_h.Repository.SampleByID(stylometry_draft.ID)
	if stylometry_err != nil {
		logrus.Error(stylometry_err)
		stylometry_ctx.JSON(http.StatusInternalServerError, gin.H{"error": "черновик опубликован, но карточку не удалось перечитать"})
		return
	}
	if !stylometry_ok {
		stylometry_ctx.JSON(http.StatusOK, gin.H{"published": true})
		return
	}

	stylometry_ctx.JSON(http.StatusOK, gin.H{
		"published": true,
		"sample":    serializer.SampleResponse(stylometry_published, stylometryAPIUserID()),
	})
}

// DELETE /api/samples/:id
func (stylometry_h *SampleHandler) APIDeleteSample(stylometry_ctx *gin.Context) {
	stylometry_id, stylometry_err := strconv.ParseUint(strings.TrimSpace(stylometry_ctx.Param("id")), 10, 64)
	if stylometry_err != nil {
		stylometry_ctx.JSON(http.StatusBadRequest, gin.H{"error": "некорректный id карточки"})
		return
	}

	stylometry_err = stylometry_h.Repository.DeleteSampleSQL(uint(stylometry_id))
	if stylometry_err != nil {
		logrus.Error(stylometry_err)
		stylometry_ctx.JSON(http.StatusNotFound, gin.H{"error": "карточка текущего пользователя не найдена или уже удалена"})
		return
	}

	stylometry_ctx.JSON(http.StatusOK, gin.H{"deleted": true})
}

// POST /api/samples/:id/like
func (stylometry_h *SampleHandler) APIPostSampleLike(stylometry_ctx *gin.Context) {
	stylometry_id, stylometry_err := strconv.ParseUint(strings.TrimSpace(stylometry_ctx.Param("id")), 10, 64)
	if stylometry_err != nil {
		stylometry_ctx.JSON(http.StatusBadRequest, gin.H{"error": "некорректный id карточки"})
		return
	}

	var stylometry_request stylometryAPILikeRequest
	if stylometry_err = stylometry_ctx.ShouldBind(&stylometry_request); stylometry_err != nil {
		stylometry_ctx.JSON(http.StatusBadRequest, gin.H{"error": "нужно передать stylometry_like: 0 или 1"})
		return
	}

	var stylometry_like_value int
	if stylometry_request.StylometryLike != nil {
		stylometry_like_value = *stylometry_request.StylometryLike
	} else if stylometry_request.Like != nil {
		stylometry_like_value = *stylometry_request.Like
	} else {
		stylometry_ctx.JSON(http.StatusBadRequest, gin.H{"error": "нужно передать stylometry_like: 0 или 1"})
		return
	}

	if stylometry_like_value != 0 && stylometry_like_value != 1 {
		stylometry_ctx.JSON(http.StatusBadRequest, gin.H{"error": "stylometry_like должен быть 0 или 1"})
		return
	}

	stylometry_ok, stylometry_err := stylometry_h.Repository.SetLike(uint(stylometry_id), stylometry_like_value)
	if stylometry_err != nil {
		logrus.Error(stylometry_err)
		stylometry_ctx.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось изменить лайк"})
		return
	}
	if !stylometry_ok {
		stylometry_ctx.JSON(http.StatusNotFound, gin.H{"error": "опубликованная карточка не найдена"})
		return
	}

	stylometry_sample, _, stylometry_err := stylometry_h.Repository.SampleByID(uint(stylometry_id))
	if stylometry_err != nil {
		logrus.Error(stylometry_err)
		stylometry_ctx.JSON(http.StatusInternalServerError, gin.H{"error": "лайк изменён, но карточку не удалось перечитать"})
		return
	}

	stylometry_ctx.JSON(http.StatusOK, gin.H{
		"liked":  stylometry_like_value == 1,
		"sample": serializer.SampleResponse(stylometry_sample, stylometryAPIUserID()),
	})
}

// POST /api/users
func (stylometry_h *SampleHandler) APIPostUserRegister(stylometry_ctx *gin.Context) {
	var stylometry_request stylometryAPIUserRequest
	if stylometry_err := stylometry_ctx.ShouldBind(&stylometry_request); stylometry_err != nil {
		stylometry_ctx.JSON(http.StatusBadRequest, gin.H{"error": "нужно передать данные пользователя"})
		return
	}

	stylometry_name := strings.TrimSpace(stylometry_request.Name)
	stylometry_password := strings.TrimSpace(stylometry_request.Password)

	if stylometry_name == "" {
		stylometry_ctx.JSON(http.StatusBadRequest, gin.H{"error": "нужно передать stylometry_sample_name"})
		return
	}
	if stylometry_password == "" {
		stylometry_password = "stylometry123"
	}

	stylometry_user, stylometry_err := stylometry_h.Repository.RegisterUser(stylometry_name, stylometry_password)
	if stylometry_err != nil {
		logrus.Error(stylometry_err)
		if strings.Contains(strings.ToLower(stylometry_err.Error()), "duplicate") {
			stylometry_ctx.JSON(http.StatusConflict, gin.H{"error": "пользователь с таким именем уже существует"})
			return
		}
		stylometry_ctx.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось зарегистрировать пользователя"})
		return
	}

	stylometry_ctx.JSON(http.StatusCreated, gin.H{
		"user": serializer.UserResponse(stylometry_user),
	})
}

// POST /api/auth/login
func (stylometry_h *SampleHandler) APIPostAuthLogin(stylometry_ctx *gin.Context) {
	stylometry_ctx.JSON(http.StatusOK, gin.H{
		"authenticated":           true,
		"stylometry_current_user": stylometryAPIUserID(),
		"stylometry_auth_comment": "заглушка для лабораторной работы №4",
	})
}

// POST /api/auth/logout
func (stylometry_h *SampleHandler) APIPostAuthLogout(stylometry_ctx *gin.Context) {
	stylometry_ctx.JSON(http.StatusOK, gin.H{
		"deauthenticated":         true,
		"stylometry_auth_comment": "заглушка для лабораторной работы №4",
	})
}
