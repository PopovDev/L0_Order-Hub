# Order Hub

Демонстрационный сервис для обработки и отображения данных о заказах в реальном времени.

Сервис подписывается на канал в NATS Streaming, получает сообщения о новых заказах, сохраняет их в нормализованном виде в PostgreSQL и кэширует в оперативной памяти для быстрой выдачи через HTTP API.

##  Демонстрация работы

Демо-видео: [https://disk.yandex.ru/i/FnWvGQKv1J3Leg](https://disk.yandex.lt/i/crSpgKtUFM4-nA)

## Быстрый старт (c использованием docker-compose)

Убедитесь, что Docker установлен и запущен.

1.  **Клонировать репозиторий:**
    ```bash
    git clone https://github.com/PopovDev/L0_Order-Hub.git
    cd L0_Order-Hub
    ```

2.  **Запустить инфраструктуру:**
    ```bash
    docker-compose up -d
    ```

3.  **Настройка базы данных:**
    ```bash
    Get-Content init.sql | docker exec -i postgres_db psql -U postgres -d orders_db
    ```

4.  **Запустить сервис:**
    ```bash
    go run cmd/service/main.go
    ```

5.  **Публикация тестового заказа (в новом терминале):**
    ```bash
    go run scripts/publisher/main.go
    ```

5.  **Открыть веб-интерфейс**
    Откройте [http://localhost:8080](http://localhost:8080) и используйте UID заказа `b563feb7b2b84b6test`.
    
