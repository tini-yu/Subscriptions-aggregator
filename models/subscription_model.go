package models

import "time"

type Subscription struct {
	ID          int        //Primary key
	ServiceName string     `json:"service_name"` // Название сервиса
	Price       int        `json:"price"`        // Стоимость в рублях
	UserID      string     `json:"user_id"`      // UUID пользователя
	StartDate   time.Time  `json:"start_date"`   // Дата начала
	EndDate     *time.Time `json:"end_date"`     // Опционально: дата окончания
}

type SubscriptionInput struct {
	ServiceName string  `json:"service_name"`
	Price       int     `json:"price"`
	UserID      string  `json:"user_id"`
	StartDate   string  `json:"start_date"` // временно строка до парсинга
	EndDate     *string `json:"end_date"`   // аналогично
}

type UpdateSubscription struct {
	ServiceName *string     `json:"service_name"`
	Price       *int        `json:"price"`
	UserID      *string     `json:"user_id"`
	StartDate   *time.Time  `json:"start_date"`
	EndDate     **time.Time `json:"end_date"`
}

type UpdateSubscriptionInput struct {
	ServiceName *string `json:"service_name,omitempty"`
	Price       *int    `json:"price,omitempty"`
	UserID      *string `json:"user_id,omitempty"`
	StartDate   *string `json:"start_date,omitempty"`
	EndDate     *string `json:"end_date,omitempty"`
}
