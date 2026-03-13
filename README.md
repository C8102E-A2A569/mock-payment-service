# Mock Payment Service

Mock-сервис платежей и переводов: заглушка с gRPC API для сценариев «создать счёт — пополнить — перевести». Подходит как зависимость для тестов или других сервисов: данные в Postgres, события в Kafka. Реальная авторизация и внешние платёжные системы не используются.

**Стек:** Go, gRPC, PostgreSQL, Kafka, Redis, Docker. Данные хранятся в Postgres, баланс кэшируется в Redis, операции можно повторять идемпотентно по ключу.

---

## Функциональность

| Метод | Описание |
|-------|----------|
| **CreateAccount** | Создаёт счёт с балансом 0 по внешнему `user_id`, возвращает `account_id` (UUID). |
| **GetBalance** | Возвращает баланс счёта: сначала из кэша Redis, при промахе — из Postgres, кэш обновляется. |
| **Deposit** | Пополнение счёта на `amount`. Пишет операцию в Postgres, публикует событие `payment.completed` в Kafka, инвалидирует/обновляет кэш. Опционально `idempotency_key` — повторный запрос с тем же ключом возвращает сохранённый результат без повторного зачисления. |
| **Transfer** | Перевод `amount` с одного счёта на другой. Выполняется в одной транзакции БД; при недостатке средств балансы не меняются, возвращается `success: false`, в Kafka уходит `transfer.failed`. При успехе — `transfer.completed`. Поддерживается идемпотентность по `idempotency_key`. Кэш баланса обновляется для обоих счетов. |

**События Kafka** (топик `payment_events`): `payment.completed` (после Deposit), `transfer.completed`, `transfer.failed` — с полями `event_id`, `type`, идентификаторами счетов/транзакций, суммой, временем.

**Redis:** кэш баланса по ключу `balance:{account_id}` (TTL несколько минут); ключи идемпотентности `idem:{idempotency_key}` с сохранённым ответом (TTL до 24 ч).

---


## Запуск

Для Docker отдельно настраивать конфиги не нужно: параметры (БД, Kafka, Redis, порт gRPC) заданы в `deployments/docker-compose.yml` через переменные окружения. Чтобы изменить порт, пароль БД или адреса — правьте `docker-compose.yml` или задайте переменные при запуске.

```bash
git clone <url-репозитория>
cd new-project

docker compose -f deployments/docker-compose.yml up -d
```


Проверка (нужен [grpcurl](https://github.com/fullstorydev/grpcurl)):

```bash
grpcurl -plaintext -d '{"user_id": "user-1"}' localhost:50051 payment.PaymentService/CreateAccount
grpcurl -plaintext -d '{"account_id": "<uuid>"}' localhost:50051 payment.PaymentService/GetBalance
grpcurl -plaintext -d '{"account_id": "<uuid>", "amount": 1000}' localhost:50051 payment.PaymentService/Deposit
grpcurl -plaintext -d '{"from_account_id": "<uuid1>", "to_account_id": "<uuid2>", "amount": 100}' localhost:50051 payment.PaymentService/Transfer
```

Остановка:

```bash
docker compose -f deployments/docker-compose.yml down
```

Kafka UI: http://localhost:8080 (просмотр топиков и сообщений).

---

## Структура репозитория

```
├── .github/workflows/ci.yml    # CI: lint, test, build, docker-образ
├── api/proto/payment/          # payment.proto и сгенерированный Go
├── cmd/
│   ├── server/main.go         # Точка входа: конфиг, БД, Kafka, Redis, gRPC
│   └── migrate/main.go        # Применение миграций
├── configs/                   # config.example.yaml (в образ копируется как config.yaml)
├── deployments/
│   ├── docker/Dockerfile
│   └── docker-compose.yml     # app, postgres, kafka, redis, zookeeper, kafka-ui
├── internal/
│   ├── config/                # Загрузка конфига (env + YAML)
│   ├── domain/                # Account, Transaction, типы операций
│   ├── repository/            # Интерфейс репозитория, Postgres, миграции
│   ├── service/               # CreateAccount, GetBalance, Deposit, Transfer
│   ├── grpc/                  # gRPC-сервер и хендлеры
│   ├── kafka/                 # Продьюсер: payment.completed, transfer.completed/failed
│   ├── cache/                 # Redis: кэш баланса, идемпотентность
│   └── testutil/              # Моки для тестов
├── pkg/apperror/              # Ошибки приложения
├── go.mod, go.sum
├── Makefile                   # docker-up, docker-down, docker-build
└── README.md
```

