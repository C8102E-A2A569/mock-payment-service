# Mock Payment Service

Заглушка платёжного сервиса с gRPC API: создание счетов, пополнение, переводы. Удобно использовать как зависимость в тестах или других сервисах.

Написано на Go, данные хранятся в PostgreSQL, баланс кэшируется в Redis, события о платежах и переводах публикуются в Kafka. Всё поднимается через Docker Compose.

---

## API

| Метод | Описание |
|-------|----------|
| CreateAccount | Создаёт счёт с нулевым балансом по идентификатору пользователя. |
| GetBalance | Возвращает баланс счёта. Читает из кэша, при промахе — из базы. |
| Deposit | Пополняет счёт. Публикует событие об успешной операции в Kafka. Поддерживает идемпотентность. |
| Transfer | Переводит сумму между двумя счётами в одной транзакции. При нехватке средств баланса не меняются, в Kafka уходит событие о неудаче. Поддерживает идемпотентность. |

---

## Запуск

```bash
git clone <url>
cd new-project
docker compose -f deployments/docker-compose.yml up -d
```

Параметры (порт, БД, Kafka, Redis) задаются через переменные окружения в `docker-compose.yml`.

Проверка (нужен [grpcurl](https://github.com/fullstorydev/grpcurl)):

```bash
grpcurl -plaintext -d '{"user_id": "user-1"}' localhost:50051 payment.PaymentService/CreateAccount
grpcurl -plaintext -d '{"account_id": "<uuid>"}' localhost:50051 payment.PaymentService/GetBalance
grpcurl -plaintext -d '{"account_id": "<uuid>", "amount": 1000}' localhost:50051 payment.PaymentService/Deposit
grpcurl -plaintext -d '{"from_account_id": "<uuid1>", "to_account_id": "<uuid2>", "amount": 100}' localhost:50051 payment.PaymentService/Transfer
```

Kafka UI: http://localhost:8090

---

## Структура

```
├── .github/workflows/ci.yml
├── api/proto/payment/
├── cmd/
│   ├── server/main.go
│   └── migrate/main.go
├── configs/
├── deployments/
│   ├── docker/Dockerfile
│   └── docker-compose.yml
├── internal/
│   ├── config/
│   ├── domain/
│   ├── repository/
│   ├── service/
│   ├── grpc/
│   ├── kafka/
│   ├── cache/
│   └── testutil/
├── pkg/apperror/
├── go.mod, go.sum
└── Makefile
```
