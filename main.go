package main

import (
	"net/http"
	"os"
	"usersubs/database"
	_ "usersubs/docs"
	"usersubs/handler"
	"usersubs/logger"
	"usersubs/service"

	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
)

// @title User Subscription Aggregator
// @version 1.0
// @description Сервер аггрегации данных об онлайн подписках пользователей.
// @host localhost:8080

func main() {
	err := logger.Init(false)
	if err != nil {
		panic("Ошибка инициализации логгера: " + err.Error())
	}
	defer logger.L().Sync()

	logger.L().Info("Запуск сервера")

	db, err := db.ConnectDB()
	if err != nil {
		logger.L().Fatal("Не удалось подключиться к БД", zap.Error(err))
	}
	defer db.Close()

	logger.L().Info("Сервер подключился к PostgreSQL")
	
	subService := service.NewSubscriptionsService(db)
	subHandler := handler.NewSubscriptionHandler(subService)

	mux := http.NewServeMux()
	mux.Handle("/swagger/", httpSwagger.WrapHandler)
	
	mux.HandleFunc("/subscriptions", subHandler.HandleSubscriptions)
	mux.HandleFunc("/subscriptions/", subHandler.HandleSubscriptionsByID)
	mux.HandleFunc("/subscriptions/total-cost", subHandler.GetTotalCost)


	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	logger.L().Info("Запуск сервера", zap.String("port", port))
	err = http.ListenAndServe(":"+port, mux)
	if err != nil {
		logger.L().Fatal("Ошибка запуска сервера", zap.Error(err))
	}
}
