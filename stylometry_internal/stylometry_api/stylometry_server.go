package api

import (
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"stylometry/stylometry_internal/stylometry_app/stylometry_database"
	"stylometry/stylometry_internal/stylometry_app/stylometry_handler"
	"stylometry/stylometry_internal/stylometry_app/stylometry_repository"
)

func StartServer() {
	log.Println("StylometrySample application start!")

	db, err := database.NewDatabase()
	if err != nil {
		logrus.Fatal("ошибка подключения к базе данных стилометрии: ", err)
	}
	repo := repository.NewRepository(db)
	sampleHandler := handler.NewSampleHandler(repo)

	router := gin.Default()
	if stylometryFrontendEnabled() {
		router.LoadHTMLGlob("stylometry_templates/*")
		router.Static("/stylometry_static", "./stylometry_static")

		router.GET("/samples", func(stylometry_ctx *gin.Context) {
			stylometry_ctx.Redirect(302, "/samples/feed")
		})
		router.GET("/samples/feed", sampleHandler.GetSampleFeed)
		router.GET("/samples/draft", sampleHandler.GetSampleDraft)
		router.GET("/samples/grid", sampleHandler.GetSampleGrid)
		router.POST("/samples/draft", sampleHandler.PostSampleCreateDraft)
		router.POST("/samples/draft/publish", sampleHandler.PostSamplePublishDraft)
		router.POST("/samples/delete", sampleHandler.PostSampleDelete)
	}

	//3 labaa
	apiRoutes := router.Group("/api")
	{
		apiRoutes.GET("/samples", sampleHandler.APIGetSampleList)
		apiRoutes.GET("/samples/feed", sampleHandler.APIGetSampleFeed)
		apiRoutes.GET("/samples/draft", sampleHandler.APIGetSampleDraft)
		apiRoutes.POST("/samples", sampleHandler.APIPostSampleDraft)
		apiRoutes.PUT("/samples/draft/publish", sampleHandler.APIPutSamplePublishDraft)
		apiRoutes.DELETE("/samples/:id", sampleHandler.APIDeleteSample)
		apiRoutes.POST("/samples/:id/like", sampleHandler.APIPostSampleLike)
		apiRoutes.POST("/users", sampleHandler.APIPostUserRegister)
		apiRoutes.POST("/auth/login", sampleHandler.APIPostAuthLogin)
		apiRoutes.POST("/auth/logout", sampleHandler.APIPostAuthLogout)
	}
	log.Println("Server start up: http://127.0.0.1:8080/api/samples")
	if err := router.Run(":8080"); err != nil {
		logrus.Fatal(err)
	}
	log.Println("Server down")
}

func stylometryFrontendEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("ENABLE_FRONTEND")), "true")
}
