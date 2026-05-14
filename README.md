# em-subscriptions

REST-сервис для агрегации данных.
```bash
docker compose up -d --build
```

После `service_healthy` для postgres сервис применит миграции и поднимет HTTP на `:8080`.

- Swagger UI:   <http://localhost:8080/swagger/index.html>
- Healthcheck:  <http://localhost:8080/healthz>
- API:          `http://localhost:8080/api/v1/subscriptions`

Остановить и удалить контейнеры (volume сохраняется):

```bash
docker compose down
```

## Локальный запуск без Docker

```bash
# подними только postgres
docker compose up -d postgres

# в отдельном терминале
go run ./cmd/api
```

Дефолтный конфиг — `./config.yaml`. Любое поле можно переопределить переменной окружения (см. `.env.example`).

## API

Base path: `/api/v1`.

| Метод  | Путь                          | Назначение |
|--------|-------------------------------|------------|
| POST   | `/subscriptions`              | Создать запись |
| GET    | `/subscriptions`              | Список с фильтрами `user_id`, `service_name`, пагинацией `limit`/`offset` |
| GET    | `/subscriptions/{id}`         | Получить по id |
| PUT    | `/subscriptions/{id}`         | Обновить |
| DELETE | `/subscriptions/{id}`         | Удалить |
| GET    | `/subscriptions/total`        | Суммарная стоимость за период |

### Формат даты

Поля `start_date`, `end_date`, `period_from`, `period_to` принимаются и отдаются как `MM-YYYY` (например `07-2025`). В БД хранятся как `DATE` (первое число месяца).

### Примеры

Создать подписку:

```bash
curl -X POST http://localhost:8080/api/v1/subscriptions \
  -H 'Content-Type: application/json' \
  -d '{
    "service_name": "Yandex Plus",
    "price": 400,
    "user_id": "60601fee-2bf1-4721-ae6f-7636e79a0cba",
    "start_date": "07-2025",
    "end_date": "12-2025"
  }'
```

Список с фильтром:

```bash
curl 'http://localhost:8080/api/v1/subscriptions?user_id=60601fee-2bf1-4721-ae6f-7636e79a0cba&limit=20'
```

Сумма за период:

```bash
curl 'http://localhost:8080/api/v1/subscriptions/total?period_from=01-2025&period_to=12-2025&user_id=60601fee-2bf1-4721-ae6f-7636e79a0cba'
```

Ответ:

```json
{
  "total": 4800,
  "period_from": "01-2025",
  "period_to": "12-2025",
  "user_id": "60601fee-2bf1-4721-ae6f-7636e79a0cba"
}
```

### Как считается `total`

Для каждой подписки, чей интервал `[start_date, end_date or +∞]` пересекается с запрошенным `[period_from, period_to]`, считаем число целых месяцев активности внутри периода:

```
lower = max(start_date, period_from)
upper = min(end_date or period_to, period_to)
months = (year(upper) - year(lower)) * 12 + (month(upper) - month(lower)) + 1
cost  = months * price
```

Сумма `cost` по всем подходящим подпискам — это и есть `total`. Расчёт выполняется одним SQL-запросом (см. `internal/repository/subscription.go:Total`).

## Структура проекта

```
.
├── cmd/api/                 # точка входа
├── internal/
│   ├── config/              # парсинг yaml + env
│   ├── handler/             # HTTP-слой + swagger-аннотации + DTO
│   ├── middleware/          # request_id, structured logger, recoverer
│   ├── logger/              # обёртка над slog
│   ├── model/               # доменные типы, включая MonthYear (MM-YYYY)
│   ├── repository/          # pgx + миграции
│   └── service/             # бизнес-логика и валидация
├── migrations/              # 000001_init.up.sql / .down.sql
├── docs/                    # сгенерированный swagger
├── config.yaml              # дефолты, переопределяются env
├── .env.example
├── Dockerfile
├── docker-compose.yml
└── Makefile
```


## Полезные команды

```bash
make run              # запуск (нужен postgres)
make build            # собрать бинарь в bin/
make test             # тесты
make swagger          # перегенерировать docs/
make docker-up        # docker compose up -d --build
make docker-logs      # логи api
make docker-down      # остановить compose-стек
```

## Тесты

Юнит-тесты для критичной для приёма логики — кастомного типа `MonthYear`:

```bash
go test ./internal/model -v
```
