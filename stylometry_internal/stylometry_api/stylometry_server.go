package api

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"stylometry/stylometry_internal/stylometry_app/stylometry_handler"
	"stylometry/stylometry_internal/stylometry_app/stylometry_repository"
)

func StartServer() {
	log.Println("Stylometry application start!")

	stylometry_stylometryRepo, stylometry_err := repository.NewStylometryRepository()
	if stylometry_err != nil {
		logrus.Fatal("ошибка инициализации репозитория стилометрии: ", stylometry_err)
	}
	stylometry_stylometryHandler := handler.NewStylometryHandler(stylometry_stylometryRepo)

	stylometry_r := gin.Default()
	stylometry_r.LoadHTMLGlob("stylometry_templates/*")
	stylometry_r.Static("/stylometry_static", "./stylometry_static")

	stylometry_r.GET("/samples", stylometry_stylometryHandler.GetStylometryFeed)
	stylometry_r.GET("/samples/:id", stylometry_stylometryHandler.GetStylometryFeedByID)
	stylometry_r.GET("/samples/draft", stylometry_stylometryHandler.GetStylometryDraft)
	stylometry_r.GET("/samples/grid", stylometry_stylometryHandler.GetStylometryGrid)

	log.Println("Server start up: http://127.0.0.1:8080/samples")
	if stylometry_err := stylometry_r.Run(":8080"); stylometry_err != nil {
		logrus.Fatal(stylometry_err)
	}
	log.Println("Server down")
}
