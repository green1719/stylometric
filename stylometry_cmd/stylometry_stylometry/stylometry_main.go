package main

import (
	"log"

	"github.com/joho/godotenv"

	"stylometry/stylometry_internal/stylometry_api"
)

// @title Stylometry API
// @version 1.0
// @description REST API текстовых образцов: список, лента, черновики, публикация, лайки и пользователи. Вход выдаёт JWT; операции создателя требуют Bearer JWT. HTML-маршруты доступны только при ENABLE_FRONTEND=true.
// @BasePath /
// @securityDefinitions.apikey BearerJWT
// @in header
// @name Authorization
// @description Токен выдаётся методом POST /api/auth/login. Передайте Authorization: Bearer <token> для защищённых операций.
func main() {
	if stylometry_err := godotenv.Load(); stylometry_err != nil {
		log.Println(".env не найден, использую переменные окружения")
	}
	api.StartServer()
}
