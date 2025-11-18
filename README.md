# Multithreaded Metrics Collector (Go + TimescaleDB)

Простой многопоточный TCP-сервер для приёма и агрегации метрик во внутреннем хранилище с периодическим сбросом в TimescaleDB.

Проект демонстрирует:

- потокобезопасные структуры данных (`sync.RWMutex`);
- асинхронный приём соединений (горутинный TCP-сервер);
- батчевую запись в TimescaleDB;
- аккуратный разбор бинарного протокола;
- юнит-тесты на ключевые компоненты;
- анализ trade-off между **latency** и **throughput**;
- небольшой `loadgen` для замера производительности.

---

## Архитектура

Высокоуровневый вид:

```text
      [Clients: TCP]
            |
            v
   +-------------------+
   |  TCP Server       |  goroutine per connection
   | (internal/server) |
   +-------------------+
            |
            v
   +-------------------+
   |  protocol.Read    |  парсит бинарный формат
   | (internal/protocol) 
   +-------------------+
            |
            v
   +-------------------+
   |  MetricStore      |  потокобезопасный in-memory буфер
   | (internal/store)  |
   +-------------------+
            |
     [RunFlusher goroutine]
            |
            v
   +-------------------+
   | TimescaleClient   |  INSERT ... VALUES ...
   | (internal/db)     |
   +-------------------+
            |
            v
   [TimescaleDB / PostgreSQL]
```
Запустим loadgen, чтобы посмотреть что получится 
```code 
go run ./cmd/loadgen -addr 127.0.0.1:9000 -metrics 200000 -concurrency 8
```
Вывод - 
```[metrics-collector] 2025/11/18 15:39:20.818426 server.go:35: listening on [::]:9000
[metrics-collector] 2025/11/18 15:39:47.670523 server.go:69: new connection from 127.0.0.1:59107
[metrics-collector] 2025/11/18 15:39:47.697234 server.go:69: new connection from 127.0.0.1:59110
[metrics-collector] 2025/11/18 15:39:47.697234 server.go:69: new connection from 127.0.0.1:59113
[metrics-collector] 2025/11/18 15:39:47.697234 server.go:69: new connection from 127.0.0.1:59111
[metrics-collector] 2025/11/18 15:39:47.697234 server.go:69: new connection from 127.0.0.1:59112
[metrics-collector] 2025/11/18 15:39:47.697234 server.go:69: new connection from 127.0.0.1:59109
[metrics-collector] 2025/11/18 15:39:47.697234 server.go:69: new connection from 127.0.0.1:59114
[metrics-collector] 2025/11/18 15:39:47.697234 server.go:69: new connection from 127.0.0.1:59108
[metrics-collector] 2025/11/18 15:39:47.875407 server.go:79: read error from 127.0.0.1:59108: EOF
[metrics-collector] 2025/11/18 15:39:47.875407 server.go:79: read error from 127.0.0.1:59113: EOF
[metrics-collector] 2025/11/18 15:39:47.878742 server.go:79: read error from 127.0.0.1:59111: EOF
[metrics-collector] 2025/11/18 15:39:47.878742 server.go:79: read error from 127.0.0.1:59107: EOF
[metrics-collector] 2025/11/18 15:39:47.878742 server.go:79: read error from 127.0.0.1:59110: EOF
[metrics-collector] 2025/11/18 15:39:47.878742 server.go:79: read error from 127.0.0.1:59114: EOF
[metrics-collector] 2025/11/18 15:39:47.884065 server.go:79: read error from 127.0.0.1:59112: EOF
[metrics-collector] 2025/11/18 15:39:47.885582 server.go:79: read error from 127.0.0.1:59109: EOF
[metrics-collector] 2025/11/18 15:39:51.435262 store.go:75: flush error: insert batch: extended protocol limited to 65535 parameters
```
Имеется проблема с разделением на батчи - все 200000 запросов оказались в одном батче, а PostgreSQL такого не позволяет

Пофиксим InsertBatch
