package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"usersubs/logger"
	"usersubs/models"
	"usersubs/service"
	"usersubs/utils"

	"go.uber.org/zap"
)

type SubscriptionHandler struct {
	Service *service.SubscriptionsService
}

func NewSubscriptionHandler(s *service.SubscriptionsService) *SubscriptionHandler {
	return &SubscriptionHandler{Service: s}
}

func (h *SubscriptionHandler) HandleSubscriptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {

	case r.Method == http.MethodGet:
		h.ListSubscriptions(w, r)

	case r.Method == http.MethodPost:
		h.CreateSubscription(w, r)

	default:
		utils.MethodNotAllowed(w, r)
	}
}

func (h *SubscriptionHandler) HandleSubscriptionsByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Получаем id из пути
	idStr := strings.TrimPrefix(r.URL.Path, "/subscriptions/")
	if idStr == "" || strings.Contains(idStr, "/") {
		logger.L().Warn("неверное id в URL")
		utils.NotFound(w, r)
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"Неправильный ID записи"}`, http.StatusBadRequest)
		return
	}

	switch {

	case r.Method == http.MethodGet:
		h.GetSubscription(w, r, id)
		return
	case r.Method == http.MethodPut:
		h.UpdateSubscription(w, r, id)
	case r.Method == http.MethodDelete:
		h.DeleteSubscription(w, r, id)

	default:
		utils.MethodNotAllowed(w, r)
	}
}

// @Summary Получить подписку по ID
// @Description Возвращает одну подписку по её ID
// @Tags Subscriptions
// @Produce json
// @Param id path int true "ID подписки"
// @Success 200 {object} models.Subscription
// @Failure 404 {object} map[string]string
// @Router /subscriptions/{id} [get]
func (h *SubscriptionHandler) GetSubscription(w http.ResponseWriter, r *http.Request, id int) {

	sub, found := h.Service.GetSubscription(id)
	if !found {
		logger.L().Warn("Подписка не найдена", zap.Int("subscription_id", id))
		http.Error(w, `{"error":"Запись не найдена"}`, http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(sub)
}

// @Summary Получить все подписки
// @Description Возвращает список всех подписок
// @Tags Subscriptions
// @Produce json
// @Success 200 {array} models.Subscription
// @Failure 500 {object} map[string]string
// @Router /subscriptions [get]
func (h *SubscriptionHandler) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	subs, err := h.Service.ListSubscriptions()
	if err != nil {
		logger.L().Error("Не удалось получить записи", zap.Error(err))
		utils.InternalServerError(w, r)
		return
	}
	json.NewEncoder(w).Encode(subs)
}

// @Summary Создать новую подписку
// @Description Добавляет новую подписку в базу данных
// @Tags Subscriptions
// @Accept json
// @Produce json
// @Param subscription body models.SubscriptionInput true "Данные подписки"
// @Success 200 {object} map[string]int
// @Failure 500 {object} map[string]string
// @Router /subscriptions [post]
func (h *SubscriptionHandler) CreateSubscription(w http.ResponseWriter, r *http.Request) {

	var input models.SubscriptionInput
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		logger.L().Error("Неправильное тело запроса", zap.Error(err))
		http.Error(w, `{"error":"Неправильное тело запроса"}`, http.StatusBadRequest)
		return
	}

	if input.ServiceName == "" || input.UserID == "" || input.StartDate == "" {
		logger.L().Warn("Отсутствуют обязательные поля")
		http.Error(w, `{"error":"Отсутствуют обязательные поля"}`, http.StatusBadRequest)
		return
	}

	startDate, err := utils.ParseMonthYear(input.StartDate)
	if err != nil {
		logger.L().Error("Неверный формат start_date", zap.Error(err))
		http.Error(w, `{"error":"Неверный формат start_date (MM-YYYY)"}`, http.StatusBadRequest)
		return
	}

	var endDate *time.Time
	if input.EndDate != nil && *input.EndDate != "" {
		t, err := utils.ParseMonthYear(*input.EndDate)
		if err != nil {
			logger.L().Error("Неверный формат end_date", zap.Error(err))
			http.Error(w, `{"error":"Неверный формат end_date (MM-YYYY)"}`, http.StatusBadRequest)
			return
		}
		endDate = &t
	}
	if endDate != nil && endDate.Before(startDate) {
		logger.L().Warn("Дата окончания не может быть раньше даты начала")
		http.Error(w, `{"error":"Дата окончания не может быть раньше даты начала"}`, http.StatusBadRequest)
		return
	}
	if input.Price < 0 {
		logger.L().Warn("Цена не может быть отрицательной")
		http.Error(w, `{"error":"Цена не может быть отрицательной"}`, http.StatusBadRequest)
		return
	}

	sub := models.Subscription{
		ServiceName: input.ServiceName,
		Price:       input.Price,
		UserID:      input.UserID,
		StartDate:   startDate,
		EndDate:     endDate,
	}
	log.Println(sub)

	id, err := h.Service.CreateSubscription(sub)
	if err != nil {
		logger.L().Error("Не удалось создать запись", zap.Error(err))
		http.Error(w, fmt.Sprintf(`{"error":"Не удалось создать запись: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	response := map[string]int{"Id": id}
	json.NewEncoder(w).Encode(response)
}

// @Summary Обновить подписку
// @Description Обновляет поля подписки по ID (только переданные и непустые поля)
// @Tags Subscriptions
// @Accept json
// @Produce json
// @Param id path int true "ID подписки"
// @Param subscription body models.UpdateSubscription true "Обновлённые поля подписки"
// @Success 200 {object} models.Subscription
// @Failure 500 {object} map[string]string
// @Router /subscriptions/{id} [put]
func (h *SubscriptionHandler) UpdateSubscription(w http.ResponseWriter, r *http.Request, id int) {

	var input models.UpdateSubscriptionInput
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		logger.L().Error("Неправильное тело запроса", zap.Error(err))
		http.Error(w, `{"error":"Неправильное тело запроса"}`, http.StatusBadRequest)
		return
	}

	var startDate *time.Time
	if input.StartDate != nil && *input.StartDate != "" {
		t, err := utils.ParseMonthYear(*input.StartDate)
		if err != nil {
			logger.L().Error("Неверный формат start_date", zap.Error(err))
			http.Error(w, `{"error":"Неверный формат start_date (MM-YYYY)"}`, http.StatusBadRequest)
			return
		}
		startDate = &t
	}

	var endDate *time.Time
	if input.EndDate != nil && *input.EndDate != "" {
		t, err := utils.ParseMonthYear(*input.EndDate)
		if err != nil {
			logger.L().Error("Неверный формат end_date", zap.Error(err))
			http.Error(w, `{"error":"Неверный формат end_date (MM-YYYY)"}`, http.StatusBadRequest)
			return
		}
		endDate = &t
	}
	if endDate != nil && endDate.Before(*startDate) {
		logger.L().Warn("Дата окончания не может быть раньше даты начала", zap.Time("start", *startDate), zap.Time("end", *endDate))
		http.Error(w, `{"error":"Дата окончания не может быть раньше даты начала"}`, http.StatusBadRequest)
		return
	}
	if *input.Price < 0 {
		logger.L().Warn("Цена не может быть отрицательной")
		http.Error(w, `{"error":"Цена не может быть отрицательной"}`, http.StatusBadRequest)
		return
	}

	sub := models.UpdateSubscription{
		ServiceName: input.ServiceName,
		Price:       input.Price,
		UserID:      input.UserID,
		StartDate:   startDate,
		EndDate:     &endDate,
	}

	updatedSub, err := h.Service.UpdateSubscription(sub, id)
	if err != nil {
		logger.L().Error("Не удалось обновить запись", zap.Error(err))
		http.Error(w, fmt.Sprintf(`{"error":"Не удалось обновить запись: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(updatedSub)
}

// @Summary Удалить подписку
// @Description Удаляет подписку по ID
// @Tags Subscriptions
// @Produce json
// @Param id path int true "ID подписки"
// @Success 204 "Подписка удалена"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /subscriptions/{id} [delete]
func (h *SubscriptionHandler) DeleteSubscription(w http.ResponseWriter, r *http.Request, id int) {

	rowsDeleted, err := h.Service.DeleteSubscription(id)
	if err != nil {
		logger.L().Error("Не удалось удалить запись", zap.Error(err))
		http.Error(w, `{"error":"Не удалось удалить запись"}`, http.StatusInternalServerError)
		return
	}
	if rowsDeleted == 0 {
		logger.L().Warn("Запись не найдена", zap.Int("Id", id))
		http.Error(w, `{"error":"Запись не найдена"}`, http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// @Summary Получение суммарной стоимости подписок
// @Description Возвращает суммарную стоимость подписок пользователя за указанный период с фильтрацией по названию сервиса (счет по месяцам)
// @Tags Subscriptions
// @Accept  json
// @Produce  json
// @Param user_id query string false "ID пользователя (UUID)"
// @Param service_name query string false "Название сервиса (опционально)"
// @Param start_date query string true "Дата начала периода (в формате MM-YYYY) (опционально)"
// @Param end_date query string true "Дата окончания периода (в формате MM-YYYY) (опционально)"
// @Success 200 {object} map[string]int "Суммарная стоимость, например {\"total_cost\": 900}"
// @Failure 400 {object} map[string]string "Ошибка валидации"
// @Failure 500 {object} map[string]string "Ошибка сервера"
// @Router /subscriptions/total-cost [get]
func (h *SubscriptionHandler) GetTotalCost(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	userID := query.Get("user_id")
	serviceName := query.Get("service_name")
	startStr := query.Get("start_date")
	endStr := query.Get("end_date")
	if startStr == "" || endStr == "" {
		logger.L().Warn("start_date, end_date обязательны")
		http.Error(w, `{"error":"start_date, end_date обязательны"}`, http.StatusBadRequest)
		return
	}

	var startDate, endDate time.Time
	var err error

	startDate, err = utils.ParseMonthYear(startStr)
	if err != nil {
		logger.L().Error("Неверный формат start_date", zap.Error(err))
		http.Error(w, `{"error":"Неверный формат start_date (MM-YYYY)"}`, http.StatusBadRequest)
		return
	}
	endDate, err = utils.ParseMonthYear(endStr)
	if err != nil {
		logger.L().Error("Неверный формат end_date", zap.Error(err))
		http.Error(w, `{"error":"Неверный формат end_date (MM-YYYY)"}`, http.StatusBadRequest)
		return
	}

	if endDate.Before(startDate) {
		logger.L().Warn("Дата окончания не может быть раньше даты начала", zap.Time("start", startDate), zap.Time("end", endDate))
		http.Error(w, `{"error":"Дата окончания не может быть раньше даты начала"}`, http.StatusBadRequest)
		return
	}

	// log.Println(userID, serviceName, startDate, endDate)
	total, err := h.Service.GetTotalCost(userID, serviceName, startDate, endDate)
	if err != nil {
		logger.L().Error("Ошибка при подсчете стоимости", zap.Error(err))
		http.Error(w, fmt.Sprintf(`{"error":"Ошибка при подсчете стоимости: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	resp := map[string]int{"total_cost": total}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
