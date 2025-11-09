package cache

import (
    "testing"
)

func TestCache_SetAndGet(t *testing.T) {
    c := New(2)

    c.Set("key1", "value1")
    value, ok := c.Get("key1")

    if !ok {
        t.Error("Ожидалось, что ключ будет найден")
    }

    if value != "value1" {
        t.Errorf("Ожидалось значение 'value1', получили '%s'", value)
    }
}

func TestCache_GetNonExistentKey(t *testing.T) {
    c := New(2)

    _, ok := c.Get("nonexistent")

    if ok {
        t.Error("Ожидалось, что ключ не будет найден")
    }
}

func TestCache_FIFO(t *testing.T) {
    c := New(3)

    c.Set("key1", "value1")
    c.Set("key2", "value2")
    c.Set("key3", "value3")

    _, ok := c.Get("key1")
    if !ok {
        t.Error("Ошибка: key1 должен быть в кэше")
    }

    c.Set("key4", "value4")

    _, ok = c.Get("key1")
    if ok {
        t.Error("Ошибка: key1 должен был быть удален из кэша по политике FIFO")
    }

    value, ok := c.Get("key4")
    if !ok || value != "value4" {
        t.Error("Ошибка: key4 должен быть в кэше с правильным значением")
    }
}

func TestCache_UpdateExistingKey(t *testing.T) {
    c := New(2)

    c.Set("key1", "value1")
    c.Set("key1", "new_value")

    value, ok := c.Get("key1")
    if !ok {
        t.Error("Ожидалось, что ключ будет найден")
    }

    if value != "new_value" {
        t.Errorf("Ожидалось значение 'new_value', получили '%s'", value)
    }

    c.Set("key2", "value2")
    _, ok = c.Get("key2")
    if !ok {
        t.Error("Ошибка: key2 должен быть в кэше")
    }
}