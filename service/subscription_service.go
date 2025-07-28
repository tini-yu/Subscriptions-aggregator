package service

import (
	"database/sql"
	"fmt"
	"log"
	"time"
	"usersubs/logger"
	"usersubs/models"

	"go.uber.org/zap"
)

type SubscriptionsService struct {
	DB *sql.DB
}

func NewSubscriptionsService(db *sql.DB) *SubscriptionsService {
	return &SubscriptionsService{DB: db}
}

func (s *SubscriptionsService) CreateSubscription(sub models.Subscription) (int, error) {
	query := `
		INSERT INTO subscriptions (service_name, price, user_id, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	log.Println("получен запрос на запись")
	var newID int
	err := s.DB.QueryRow(query,
		sub.ServiceName,
		sub.Price,
		sub.UserID,
		sub.StartDate,
		sub.EndDate,
	).Scan(&newID)

	if err != nil {
		return 0, err
	}

	return newID, nil
}

func (s *SubscriptionsService) UpdateSubscription(sub models.UpdateSubscription, id int) (models.Subscription, error) {

	//Ищем существующую запись
	var existing models.Subscription
	query := `SELECT id, service_name, price, user_id, start_date, end_date FROM subscriptions WHERE id = $1`
	row := s.DB.QueryRow(query, id)
	err := row.Scan(&existing.ID, &existing.ServiceName, &existing.Price, &existing.UserID, &existing.StartDate, &existing.EndDate)
	if err != nil {
		return models.Subscription{}, err
	}
	// Обновляем только указанные в теле, непустые записи
	if sub.ServiceName != nil && *sub.ServiceName != "" {
		existing.ServiceName = *sub.ServiceName
	}
	if sub.Price != nil && *sub.Price != 0 {
		existing.Price = *sub.Price
	}
	if sub.UserID != nil && *sub.UserID != "" {
		existing.UserID = *sub.UserID
	}
	if sub.StartDate != nil {
		existing.StartDate = *sub.StartDate
	}
	if sub.EndDate != nil {
		// Даже если *sub.EndDate == nil —> устанавливаем NULL
		existing.EndDate = *sub.EndDate
	}

	row = s.DB.QueryRow(`
		UPDATE subscriptions SET 
			service_name = $1,
			price = $2,
			user_id = $3,
			start_date = $4,
			end_date = $5
		WHERE id = $6
		RETURNING id, service_name, price, user_id, start_date, end_date
	`,
		existing.ServiceName,
		existing.Price,
		existing.UserID,
		existing.StartDate,
		existing.EndDate,
		id,
	)

	var updated models.Subscription
	err = row.Scan(&updated.ID, &updated.ServiceName, &updated.Price, &updated.UserID, &updated.StartDate, &updated.EndDate)
	if err != nil {
		return models.Subscription{}, err
	}
	return updated, nil
}

func (s *SubscriptionsService) GetSubscription(id int) (models.Subscription, bool) {
	var sub models.Subscription
	var endDate sql.NullTime

	query := `
		SELECT id, service_name, price, user_id, start_date, end_date
		FROM subscriptions
		WHERE id = $1
	`

	row := s.DB.QueryRow(query, id)
	err := row.Scan(
		&sub.ID,
		&sub.ServiceName,
		&sub.Price,
		&sub.UserID,
		&sub.StartDate,
		&endDate,
	)

	if err != nil {
		//возвращает пустой struct, если нет строки с данным id
		if err == sql.ErrNoRows {
			return sub, false
		}
		log.Println("GetSubscription error:", err)
		return sub, false
	}

	if endDate.Valid {
		sub.EndDate = &endDate.Time
	} else {
		sub.EndDate = nil
	}

	return sub, true
}

func (s *SubscriptionsService) ListSubscriptions() ([]models.Subscription, error) {
	rows, err := s.DB.Query("SELECT id, service_name, price, user_id, start_date, end_date FROM subscriptions")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []models.Subscription

	for rows.Next() {
		var sub models.Subscription
		var endDate sql.NullTime

		err := rows.Scan(&sub.ID, &sub.ServiceName, &sub.Price, &sub.UserID, &sub.StartDate, &endDate)
		if err != nil {
			log.Println("Scan error:", err)
			continue
		}
		//Добавляем дату окончания в struct, если существует
		if endDate.Valid {
			sub.EndDate = &endDate.Time
		} else {
			sub.EndDate = nil
		}

		subs = append(subs, sub)
	}

	return subs, nil
}

func (s *SubscriptionsService) DeleteSubscription(id int) (int64, error) {

	query := `DELETE FROM subscriptions WHERE id = $1`
	result, err := s.DB.Exec(query, id)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

func (s *SubscriptionsService) GetTotalCost(userID string, serviceName string,
	startDate time.Time, endDate time.Time) (int, error) {
	query := `
		SELECT price, start_date, end_date
		FROM subscriptions
		WHERE (start_date <= $2) AND (end_date >= $1 OR end_date IS NULL)
	`
	//Интерфейс для дополнения sql query другими агрументами, если существуют
	args := []interface{}{startDate, endDate}
	argID := 3 //счетчик позиции для след. аргументов в sql query

	if serviceName != "" {
		query += fmt.Sprintf(" AND service_name ILIKE $%d", argID)
		args = append(args, "%"+serviceName+"%")
		argID++
	}
	if userID != "" {
		query += fmt.Sprintf(" AND user_id = $%d", argID)
		args = append(args, "%"+userID+"%")
		argID++
	}

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	// Посчитать сколько целых месяцев в промежутке, и сколько всего сумма подписок в эти месяца включительно:
	total := 0

	for rows.Next() {
		var price int
		var subStart, subEnd *time.Time
		err := rows.Scan(&price, &subStart, &subEnd)
		if err != nil {
			return 0, err
		}

		actualStart := maxTime(*subStart, startDate)
		actualEnd := endDate
		if subEnd != nil && subEnd.Before(endDate) {
			actualEnd = *subEnd
		}

		// Целое кол-во месяцев:
		months := (actualEnd.Year()-actualStart.Year())*12 + int(actualEnd.Month()-actualStart.Month()) + 1
		if months > 0 {
			total += months * price
		}
		logger.L().Info("Целое кол-во месяцев", zap.Int("Months", months))
	}

	return total, nil
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
