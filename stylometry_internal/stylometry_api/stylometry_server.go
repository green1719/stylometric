package api

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"stylometry/stylometry_internal/stylometry_app/stylometry_database"
	"stylometry/stylometry_internal/stylometry_app/stylometry_handler"
	"stylometry/stylometry_internal/stylometry_app/stylometry_repository"
)

func StartServer() {
	log.Println("StylometrySample application start!")

	stylometry_db, stylometry_err := database.NewDatabase()
	if stylometry_err != nil {
		logrus.Fatal("ошибка подключения к базе данных стилометрии: ", stylometry_err)
	}
	stylometry_stylometryRepo := repository.NewStylometrySampleRepository(stylometry_db)
	stylometry_stylometryHandler := handler.NewStylometrySampleHandler(stylometry_stylometryRepo)

	stylometry_r := gin.Default()
	stylometry_r.LoadHTMLGlob("stylometry_templates/*")
	stylometry_r.Static("/stylometry_static", "./stylometry_static")

	stylometry_r.GET("/samples", func(stylometry_ctx *gin.Context) {
		stylometry_ctx.Redirect(302, "/samples/feed")
	})
	stylometry_r.GET("/samples/feed", stylometry_stylometryHandler.GetStylometrySampleFeed)
	stylometry_r.GET("/samples/draft", stylometry_stylometryHandler.GetStylometrySampleDraft)
	stylometry_r.GET("/samples/grid", stylometry_stylometryHandler.GetStylometrySampleGrid)
	stylometry_r.POST("/samples/draft", stylometry_stylometryHandler.PostStylometrySampleCreateDraft)
	stylometry_r.POST("/samples/draft/publish", stylometry_stylometryHandler.PostStylometrySamplePublishDraft)
	stylometry_r.POST("/samples/delete", stylometry_stylometryHandler.PostStylometrySampleDelete)

	//3 labaa
	stylometry_api := stylometry_r.Group("/api")
	{
		stylometry_api.GET("/samples", stylometry_stylometryHandler.APIGetStylometrySampleList)
		stylometry_api.GET("/samples/feed", stylometry_stylometryHandler.APIGetStylometrySampleFeed)
		stylometry_api.GET("/samples/draft", stylometry_stylometryHandler.APIGetStylometrySampleDraft)
		stylometry_api.POST("/samples", stylometry_stylometryHandler.APIPostStylometrySampleDraft)
		stylometry_api.PUT("/samples/draft/publish", stylometry_stylometryHandler.APIPutStylometrySamplePublishDraft)
		stylometry_api.DELETE("/samples/:id", stylometry_stylometryHandler.APIDeleteStylometrySample)
		stylometry_api.POST("/samples/:id/like", stylometry_stylometryHandler.APIPostStylometrySampleLike)
		stylometry_api.POST("/users", stylometry_stylometryHandler.APIPostStylometryUserRegister)
		stylometry_api.POST("/auth/login", stylometry_stylometryHandler.APIPostStylometryAuthLogin)
		stylometry_api.POST("/auth/logout", stylometry_stylometryHandler.APIPostStylometryAuthLogout)
	}
	log.Println("Server start up: http://127.0.0.1:8080/samples/feed")
	if stylometry_err := stylometry_r.Run(":8080"); stylometry_err != nil {
		logrus.Fatal(stylometry_err)
	}
	log.Println("Server down")
}
