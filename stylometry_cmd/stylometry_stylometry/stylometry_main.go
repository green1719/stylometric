package main

import (
	"log"

	"github.com/joho/godotenv"

	"stylometry/stylometry_internal/stylometry_api"
)

func main() {
	if stylometry_err := godotenv.Load(); stylometry_err != nil {
		log.Println(".env не найден, использую значения по умолчанию")
	}
	api.StartServer()
}
