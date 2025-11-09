package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gorilla/mux"
    "github.com/nats-io/stan.go"
    "wb-order-hub/internal/cache"
    "wb-order-hub/internal/config"
    "wb-order-hub/internal/database"
    "wb-order-hub/internal/dto"
    "wb-order-hub/internal/models"
)

var (
    orderCache *cache.Cache
    db         *sql.DB
    sc         stan.Conn
)

func main() {
    cfg := config.Load()

    dbConfig := database.DBConfig{
        Host:     cfg.DatabaseHost,
        Port:     cfg.DatabasePort,
        User:     cfg.DatabaseUser,
        Password: cfg.DatabasePassword,
        DBName:   cfg.DatabaseName,
    }
    var err error
    db, err = database.NewDBConnection(dbConfig)
    if err != nil {
        log.Fatalf("Не удалось подключиться к базе данных: %v", err)
    }
    defer db.Close()

    orderCache = cache.New(100)
    restoreCache()

    sc, err = stan.Connect(cfg.NatsClusterID, cfg.NatsClientID, stan.NatsURL(cfg.NatsURL))
    if err != nil {
        log.Fatalf("Не удалось подключиться к NATS Streaming: %v", err)
    }
    defer sc.Close()

    subscribeToOrders(sc)

    router := mux.NewRouter()
    router.HandleFunc("/order/{id}", getOrderHandler).Methods("GET")
    router.PathPrefix("/").Handler(http.FileServer(http.Dir("web/")))

    srv := &http.Server{
        Addr:         ":" + cfg.ServerPort,
        Handler:      router,
        WriteTimeout: 15 * time.Second,
        ReadTimeout:  15 * time.Second,
    }

    go func() {
        log.Printf("Запуск HTTP сервера на порту :%s...", cfg.ServerPort)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Сервер не смог запуститься: %v", err)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    log.Println("Получен сигнал завершения. Начинаю корректную остановку...")

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Printf("Ошибка при остановке сервера: %v", err)
    }

    log.Println("Сервис успешно остановлен.")
}

func restoreCache() {
    log.Println("Восстановление кэша из базы данных...")
    orders, err := database.GetAllOrders(db)
    if err != nil {
        log.Printf("Предупреждение: не удалось восстановить кэш из БД: %v", err)
        return
    }
    for uid, order := range orders {
        jsonOrder, _ := json.Marshal(order)
        orderCache.Set(uid, string(jsonOrder))
    }
    log.Printf("Кэш восстановлен, заказов в кэше: %d.", len(orders))
}

func subscribeToOrders(sc stan.Conn) {
    log.Println("Подписка на NATS канал 'orders'...")
    _, err := sc.Subscribe("orders", func(m *stan.Msg) {
        log.Printf("Получено сообщение: %s", string(m.Data))

        var order models.Order
        if err := json.Unmarshal(m.Data, &order); err != nil {
            log.Printf("Ошибка десериализации сообщения: %v", err)
            return
        }

        if order.OrderUID == "" {
            log.Printf("Получено сообщение с пустым order_uid, пропускаем")
            return
        }

        if err := database.SaveOrder(db, order); err != nil {
            log.Printf("Не удалось сохранить заказ %s в БД: %v", order.OrderUID, err)
            return
        }

        jsonOrder, _ := json.Marshal(order)
        orderCache.Set(order.OrderUID, string(jsonOrder))
        log.Printf("Заказ %s обработан и добавлен в кэш", order.OrderUID)

    }, stan.DurableName("order-service-durable"))

    if err != nil {
        log.Fatalf("Не удалось подписаться на NATS канал: %v", err)
    }
}

func getOrderHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    orderID := vars["id"]

    if orderID == "" {
        http.Error(w, "ID заказа обязателен", http.StatusBadRequest)
        return
    }

    value, ok := orderCache.Get(orderID)
    if !ok {
        http.Error(w, "Заказ не найден", http.StatusNotFound)
        return
    }

    var orderModel models.Order
    if err := json.Unmarshal([]byte(value), &orderModel); err != nil {
        http.Error(w, "Ошибка обработки данных заказа", http.StatusInternalServerError)
        return
    }

    orderResponse := dto.ToResponse(orderModel)

    responseJson, err := json.Marshal(orderResponse)
    if err != nil {
        http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write(responseJson)
}