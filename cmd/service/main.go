package main

import (
    "database/sql"
    "encoding/json"
    "log"
    "net/http"
    "time"

    "github.com/gorilla/mux"
    "github.com/nats-io/stan.go"
    "wb-order-hub/internal/cache"
    "wb-order-hub/internal/database"
    "wb-order-hub/internal/models"
)

var (
    orderCache *cache.Cache
    db         *sql.DB
)

func main() {
    dbConfig := database.DBConfig{
        Host:     "127.0.0.1",
        Port:     "5433",
        User:     "postgres",
        Password: "121212",
        DBName:   "orders_db",
    }
    var err error
    db, err = database.NewDBConnection(dbConfig)
    if err != nil {
        log.Fatalf("Не удалось подключиться к базе данных: %v", err)
    }
    defer db.Close()

    orderCache = cache.New(100)

    restoreCache()

    sc, err := stan.Connect("test-cluster", "order-service-sub", stan.NatsURL("nats://localhost:4222"))
    if err != nil {
        log.Fatalf("Не удалось подключиться к NATS Streaming: %v", err)
    }
    defer sc.Close()

    subscribeToOrders(sc)

    startHTTPServer()
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

func startHTTPServer() {
    log.Println("Запуск HTTP сервера на порту :8080...")
    r := mux.NewRouter()

    r.HandleFunc("/order/{id}", getOrderHandler).Methods("GET")
    r.PathPrefix("/").Handler(http.FileServer(http.Dir("web/")))

    srv := &http.Server{
        Handler:      r,
        Addr:         "127.0.0.1:8080",
        WriteTimeout: 15 * time.Second,
        ReadTimeout:  15 * time.Second,
    }

    log.Fatal(srv.ListenAndServe())
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

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(value))
}