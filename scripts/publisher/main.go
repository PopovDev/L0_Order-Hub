package main

import (
    "log"
    "os"

    "github.com/nats-io/stan.go"
)

func main() {
    jsonData, err := os.ReadFile("model.json")
    if err != nil {
        log.Fatalf("Ошибка при чтении файла model.json: %v", err)
    }

    sc, err := stan.Connect("test-cluster", "order-service-publisher", stan.NatsURL("nats://localhost:4222"))
    if err != nil {
        log.Fatalf("Не удалось подключиться к NATS Streaming: %v", err)
    }
    defer sc.Close()

    log.Println("Публикация сообщения в канал 'orders'...")
    err = sc.Publish("orders", jsonData)
    if err != nil {
        log.Fatalf("Не удалось опубликовать сообщение: %v", err)
    }

    log.Println("Сообщение успешно опубликовано!")
}