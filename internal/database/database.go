package database

import (
    "database/sql"
    "fmt"
    "log"
    _ "github.com/jackc/pgx/v5/stdlib"
    "wb-order-hub/internal/models"
)

type DBConfig struct {
    Host     string
    Port     string
    User     string
    Password string
    DBName   string
}

func NewDBConnection(cfg DBConfig) (*sql.DB, error) {
    dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName)

    db, err := sql.Open("pgx", dsn)
    if err != nil {
        return nil, fmt.Errorf("не удалось подключиться к базе данных: %w", err)
    }

    if err = db.Ping(); err != nil {
        return nil, fmt.Errorf("проверка связи с базой данных не удалась: %w", err)
    }

    log.Println("Успешное подключение к базе данных!")
    return db, nil
}

// SaveOrder сохраняет полный заказ в БД в одной транзакции.
func SaveOrder(db *sql.DB, order models.Order) error {
    tx, err := db.Begin()
    if err != nil {
        return fmt.Errorf("не удалось начать транзакцию: %w", err)
    }
    defer tx.Rollback()

    _, err = tx.Exec(`
        INSERT INTO orders (order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
        ON CONFLICT (order_uid) DO NOTHING`,
        order.OrderUID, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature,
        order.CustomerID, order.DeliveryService, order.Shardkey, order.SmID, order.DateCreated, order.OofShard,
    )
    if err != nil {
        return fmt.Errorf("не удалось вставить заказ: %w", err)
    }

    _, err = tx.Exec(`
        INSERT INTO delivery (order_uid, name, phone, zip, city, address, region, email)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        ON CONFLICT (order_uid) DO NOTHING`,
        order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip,
        order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email,
    )
    if err != nil {
        return fmt.Errorf("не удалось вставить данные о доставке: %w", err)
    }

    _, err = tx.Exec(`
        INSERT INTO payment (order_uid, transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
        ON CONFLICT (order_uid) DO NOTHING`,
        order.OrderUID, order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency,
        order.Payment.Provider, order.Payment.Amount, order.Payment.PaymentDt, order.Payment.Bank,
        order.Payment.DeliveryCost, order.Payment.GoodsTotal, order.Payment.CustomFee,
    )
    if err != nil {
        return fmt.Errorf("не удалось вставить данные об оплате: %w", err)
    }

    for _, item := range order.Items {
        _, err = tx.Exec(`
            INSERT INTO items (order_uid, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
            order.OrderUID, item.ChrtID, item.TrackNumber, item.Price, item.RID,
            item.Name, item.Sale, item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status,
        )
        if err != nil {
            return fmt.Errorf("не удалось вставить товар: %w", err)
        }
    }

    if err = tx.Commit(); err != nil {
        return fmt.Errorf("не удалось подтвердить транзакцию: %w", err)
    }

    log.Printf("Заказ %s успешно сохранен", order.OrderUID)
    return nil
}

// GetAllOrders загружает все заказы из БД для восстановления кэша.
func GetAllOrders(db *sql.DB) (map[string]models.Order, error) {
    orders := make(map[string]models.Order)

    rows, err := db.Query("SELECT order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard FROM orders")
    if err != nil {
        return nil, fmt.Errorf("не удалось выполнить запрос к заказам: %w", err)
    }
    defer rows.Close()

    for rows.Next() {
        var order models.Order
        err := rows.Scan(&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale,
            &order.InternalSignature, &order.CustomerID, &order.DeliveryService,
            &order.Shardkey, &order.SmID, &order.DateCreated, &order.OofShard)
        if err != nil {
            return nil, fmt.Errorf("не удалось просканировать данные заказа: %w", err)
        }
        orders[order.OrderUID] = order
    }

    for uid, order := range orders {
        order.Delivery, _ = getDelivery(db, uid)
        order.Payment, _ = getPayment(db, uid)
        order.Items, _ = getItems(db, uid)
        orders[uid] = order
    }

    log.Printf("Загружено %d заказов из базы данных", len(orders))
    return orders, nil
}

func getDelivery(db *sql.DB, orderUID string) (models.Delivery, error) {
    var d models.Delivery
    err := db.QueryRow("SELECT name, phone, zip, city, address, region, email FROM delivery WHERE order_uid = $1", orderUID).
        Scan(&d.Name, &d.Phone, &d.Zip, &d.City, &d.Address, &d.Region, &d.Email)
    return d, err
}

func getPayment(db *sql.DB, orderUID string) (models.Payment, error) {
    var p models.Payment
    err := db.QueryRow("SELECT transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee FROM payment WHERE order_uid = $1", orderUID).
        Scan(&p.Transaction, &p.RequestID, &p.Currency, &p.Provider, &p.Amount, &p.PaymentDt, &p.Bank, &p.DeliveryCost, &p.GoodsTotal, &p.CustomFee)
    return p, err
}

func getItems(db *sql.DB, orderUID string) ([]models.Item, error) {
    rows, err := db.Query("SELECT chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status FROM items WHERE order_uid = $1", orderUID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var items []models.Item
    for rows.Next() {
        var item models.Item
        if err := rows.Scan(&item.ChrtID, &item.TrackNumber, &item.Price, &item.RID, &item.Name, &item.Sale, &item.Size, &item.TotalPrice, &item.NmID, &item.Brand, &item.Status); err != nil {
            return nil, err
        }
        items = append(items, item)
    }
    return items, nil
}