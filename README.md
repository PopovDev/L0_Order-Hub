# Order Hub

Демонстрационный сервис для обработки и отображения данных о заказах в реальном времени.

Сервис подписывается на канал в NATS Streaming, получает сообщения о новых заказах, сохраняет их в нормализованном виде в PostgreSQL и кэширует в оперативной памяти для быстрой выдачи через HTTP API.

## Технологический стек

-   Go
-   PostgreSQL
-   NATS Streaming
-   Docker & Docker Compose
-   HTML5, CSS3, JavaScript

##  Демонстрация работы

Демо-видео: [https://disk.yandex.ru/i/FnWvGQKv1J3Leg](https://disk.yandex.lt/i/crSpgKtUFM4-nA)
