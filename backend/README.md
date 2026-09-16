# AI Hackathon — Real Estate Platform & AI Agent Integration

Платформа для девелоперов и покупателей недвижимости с каталогом ЖК, квартир и ходом строительства, чатами с менеджерами, системой ведения сделок и интеграцией с ИИ-агентом (GigaChat) через шину сообщений RabbitMQ.

---

## 🏛 Архитектура системы

Платформа построена на микросервисной/модульной архитектуре:

1. **PostgreSQL** — основная реляционная база данных (миграции инициализируются автоматически из `./backend/migrations`).
2. **RabbitMQ** — брокер сообщений для асинхронного RPC-взаимодействия с **Agent Service** (GigaChat).
3. **Go User Server (`:8080`)** — клиентский HTTP API:
   - Регистрация и авторизация клиентов (JWT, Cookie / Bearer).
   - Каталог жилых комплексов, корпусов, квартир и хода строительства.
   - Создание заявок и чат-сессий (как для зарегистрированных пользователей, так и для гостей).
   - Чат с менеджерами.
   - Просмотр своих предложений и сделок.
4. **Go Staff Server (`:8081`)** — корпоративный HTTP API для менеджеров и супервайзеров:
   - Управление объектами недвижимости (ЖК, здания, квартиры, этапы стройки).
   - Обработка входящих обращений и заявок (взятие в работу, переписка, закрытие, отклонение с указанием причины).
   - Формирование коммерческих предложений и оформление сделок с расчетом скидок.
   - Доступ к AI-ассистенту для генерации альтернативных предложений клиентам.
5. **Agent Service (AI / GigaChat)** — внешний сервис ИИ-агентов, взаимодействующий с Go Backend через exchange `app.topic` по протоколу RPC.

---

## 📋 Реализованные сценарии

- **Сценарий 1 (Зарегистрированный клиент):** Клиент выбирает квартиру в каталоге, оставляет заявку/открывает чат (`POST /api/chat/sessions`), менеджер берет обращение в работу (`take`), согласует детали и оформляет сделку (`deals`).
- **Сценарий 2 (Гостевой лид):** Неавторизованный клиент выбирает квартиру, указывает телефон/email в заявке; менеджер видит контактные данные в списке сессий и связывается для закрытия сделки.
- **Сценарий 3 (Консультация по объекту):** Клиент задает вопрос в чате конкретной квартиры, менеджер отвечает, закрывает обращение либо переводит в заявку на покупку.
- **Сценарий 4 (Отказ и подбор альтернатив через ИИ):** Если клиент отказывается от покупки, менеджер фиксирует отказ с причиной (`POST /api/chat/sessions/{id}/reject`), запрашивает у ИИ персонализированные альтернативы (`POST /api/ai/chat`) и направляет новое КП.
- **Сценарий 5 (Помощь сотруднику):** менеджер анализирует переписку и готовит внутренний черновик ответа через AI. Клиент получает только итоговое сообщение менеджера; прямого доступа к AI у клиента нет.

---

## 🗄 Структура базы данных

### Основные сущности

- **`users`**: пользователи платформы (роли: `user`, `manager`, `supervisor`).
- **`residential_complexes`**: жилые комплексы (название, адрес, описание).
- **`buildings`**: корпуса домов (этажность, координаты, сроки сдачи, материал стен, статус: `design`, `construction`, `completed`, `suspended`).
- **`apartments`**: квартиры (номер, комнатность, этаж, площадь, базовая цена, отделка: `rough`, `white_box`, `turnkey`, статус: `free`, `booked`, `sold`).
- **`construction_progress`**: динамика строительства корпусов (этапы: `excavation`, `foundation`, `frame`, `roofing`, `finishing`, процент готовности, причины задержек).
- **`chat_sessions`**: сессии диалогов и заявки (`id_user`, `id_employee`, `id_apartment`, контакты гостей, статус: `open`, `in_progress`, `close`).
- **`chat_session_rejections`**: причины отказов по заявкам (`id_chat_sessions`, `id_employee`, `reason`, `created_at`).
- **`messages`**: сообщения чата (`id_chat_session`, `id_user`, `sender_type`: `client`, `manager`, `ai`, `system`, `content`, `is_read`, `sended_at`).
- **`deals`**: коммерческие предложения и сделки (`id_user`, `id_employee`, `id_apartment`, `base_price`, `percent_discount`, `total_price`, статус: `pending`, `contract`, `completed`, `cancelled`).

---

## 🚀 Запуск проекта

### 1. Переменные окружения (`backend/.env`)

Создайте или отредактируйте файл `backend/.env`:

```env
DB_HOST=database
DB_PORT=5432
DB_USER=postgres_user
DB_PASS=postgres_password
DB_NAME=dsk
JWT_SECRET=supersecretjwt
PORT=8080
RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/
MAX_MANAGER_DISCOUNT_PERCENT=5
MAX_SUPERVISOR_DISCOUNT_PERCENT=15

# Настройки отправки почты через SMTP (уведомления об изменении сроков/рисках)
SMTP_HOST=smtp.yandex.ru
SMTP_PORT=587
SMTP_USERNAME=notifications@dsk-agent.ru
SMTP_PASSWORD=your_app_password
SMTP_FROM=notifications@dsk-agent.ru
```


### 2. Запуск через Docker Compose

```bash
docker-compose up --build
```

### 2.1 Полные демонстрационные данные

После запуска контейнеров примените forward-миграции и идемпотентный seed:

```bash
sh backend/scripts/seed-demo.sh
```

Демонстрационные аккаунты используют пароль `Demo123!`:

| Роль | Логин |
|---|---|
| Руководитель отдела | `supervisor@dsk.demo` |
| Менеджер продаж | `manager@dsk.demo` |
| Второй менеджер | `manager2@dsk.demo` |
| Клиент | `anna.smirnova@example.demo` |

Seed содержит клиентов, объекты и квартиры, этапы стройки, ERP-события, остатки,
конкурентов, чаты, сделки, КП и согласования во всех поддерживаемых состояниях.
Повторный запуск обновляет демонстрационный набор и не создаёт дубликаты.

Сервисы будут доступны по следующим адресам:
- **User Server**: `http://localhost:8080`
- **Staff Server**: `http://localhost:8081`
- **PostgreSQL**: `localhost:5432`
- **RabbitMQ**: `localhost:5672` (Web UI управления: `http://localhost:15672`, логин: `guest`, пароль: `guest`)

### 3. Локальный запуск без Docker

Убедитесь, что запущены PostgreSQL и RabbitMQ:

```bash
cd backend

# Запуск клиентского сервера (порт 8080)
go run ./cmd/user_server/main.go

# Запуск сервера для сотрудников (порт 8081)
go run ./cmd/staff_server/main.go
```

---

## 📖 Swagger Документация (Интерактивный UI)

Оба сервера содержат встроенный интерактивный Swagger UI с поддержкой авторизации по JWT Bearer токену, предпросмотром схем и тестированием запросов в реальном времени:

- **Клиентский API (User Server)**:
  - **Swagger UI**: [http://localhost:8080/swagger](http://localhost:8080/swagger)
  - **OpenAPI JSON спецификация**: [http://localhost:8080/swagger/doc.json](http://localhost:8080/swagger/doc.json)
  - **Файл спецификации**: [`backend/docs/user-api-swagger.json`](file:///c:/Projects/AI-hackathon/backend/docs/user-api-swagger.json)
- **Корпоративный API (Staff Server)**:
  - **Swagger UI**: [http://localhost:8081/swagger](http://localhost:8081/swagger)
  - **OpenAPI JSON спецификация**: [http://localhost:8081/swagger/doc.json](http://localhost:8081/swagger/doc.json)
  - **Файл спецификации**: [`backend/docs/staff-api-swagger.json`](file:///c:/Projects/AI-hackathon/backend/docs/staff-api-swagger.json)

Файлы спецификаций можно напрямую импортировать в **Postman**, **Insomnia**, **Swagger Editor** или использовать для автогенерации SDK фронтенда.

---

## 📡 Спецификация HTTP API

### 1. Аутентификация (Общая)

| Метод | URL | Описание |
|---|---|---|
| `POST` | `/api/auth/register` | Регистрация клиента (на user-server) или сотрудника (на staff-server) |
| `POST` | `/api/auth/login` | Вход в систему (возвращает JWT токен и устанавливает HTTP-only cookie) |
| `POST` | `/api/auth/logout` | Выход из системы |
| `GET` | `/api/auth/profile` | Текущий профиль пользователя |

---

### 2. Каталог и ход строительства (User Server `:8080` / Staff Server `:8081`)

| Метод | URL | Сервер | Описание |
|---|---|---|---|
| `GET` | `/api/complexes` | Оба | Список всех ЖК |
| `GET` | `/api/complexes/{id}` | Оба | Детали конкретного ЖК |
| `GET` | `/api/complexes/{complexId}/buildings` | Оба | Корпуса конкретного ЖК |
| `GET` | `/api/buildings/{id}` | Оба | Детали корпуса |
| `GET` | `/api/buildings/{buildingId}/apartments` | Оба | Квартиры в выбранном корпусе |
| `GET` | `/api/buildings/{buildingId}/progress` | Оба | Динамика строительства корпуса |
| `GET` | `/api/apartments/{id}` | Оба | Детали квартиры |
| `POST/PUT/DELETE` | `/api/complexes`, `/api/buildings`, `/api/apartments`, `/api/progress` | Staff | CRUD-операции для сотрудников |

---

### 3. Чаты и заявки

#### Клиентский сервер (`:8080`):
- `POST /api/chat/sessions` — создание заявки/чата. Работает как для авторизованного клиента, так и для гостей.
  ```json
  {
    "id_apartment": 42,
    "guest_name": "Иван",
    "guest_phone": "+79991234567",
    "guest_email": "ivan@example.com",
    "message": "Здравствуйте! Хочу уточнить условия рассрочки."
  }
  ```
- `GET /api/chat/sessions` — список обращений текущего клиента.
- `GET /api/chat/sessions/{id}` — детали сессии.
- `GET /api/chat/sessions/{id}/messages` — список сообщений в чате (автоматически помечает прочитанными).
- `POST /api/chat/sessions/{id}/messages` — отправить сообщение:
  ```json
  {
    "content": "Подскажите, возможна ли военная ипотека?"
  }
  ```

#### Сервер сотрудников (`:8081`):
- `GET /api/chat/sessions?status=open` — просмотр очереди заявок (`open`, `in_progress`, `close`).
- `GET /api/chat/sessions/{id}` — просмотр деталей заявки и контактов клиента.
- `POST /api/chat/sessions/{id}/take` — взять заявку в работу (назначает текущего менеджера и меняет статус на `in_progress`).
- `POST /api/chat/sessions/{id}/close` — закрыть обращение (`status: close`).
- `POST /api/chat/sessions/{id}/reject` — зафиксировать отказ клиента с причиной:
  ```json
  {
    "reason": "Клиент выбрал другой район, не подошла этажность"
  }
  ```
- `GET /api/chat/sessions/{id}/messages` — история сообщений диалога.
- `POST /api/chat/sessions/{id}/messages` — ответ менеджера клиенту.

---

### 4. Сделки и коммерческие предложения

#### Клиентский сервер (`:8080`):
- `GET /api/deals` — список предложений и сделок клиента.
- `GET /api/deals/{id}` — детали сделки (базовая стоимость, скидка, итоговая цена, статус).

#### Сервер сотрудников (`:8081`):
- `POST /api/deals` — создать сделку / сформировать предложение:
  ```json
  {
    "id_user": 10,
    "id_apartment": 42,
    "base_price": 12500000.00,
    "percent_discount": 5.0
  }
  ```
  *(Поле `total_price` рассчитывается автоматически: `11 875 000.00`)*.
- `GET /api/deals` — список всех сделок компании (фильтры `status`, `user_id`).
- `GET /api/deals/{id}` — детали сделки.
- `PUT /api/deals/{id}/status` — обновить статус (`pending`, `contract`, `completed`, `cancelled`) и/или размер скидки:
  ```json
  {
    "status": "contract",
    "percent_discount": 7.5
  }
  ```

---

### 5. Интеграция с AI Agent (GigaChat) через RabbitMQ

Эндпоинт доступен только сотрудникам на Staff Server:
- `POST /api/ai/chat`

#### Пример запроса от Frontend:
```json
{
  "message": "Привет! Расскажи, чем ты можешь помочь менеджеру?",
  "session_id": "1",
  "deal_id": null
}
```
*Примечание: backend автоматически передаёт ID сотрудника из токена. Внутренние AI-рекомендации не публикуются в клиентскую переписку автоматически.*

#### Транспортный RPC-путь:
1. `Frontend` → `POST /api/ai/chat` (Go Backend)
2. `Go Backend` сериализует сообщение и отправляет в RabbitMQ:
   - **Exchange**: `app.topic`
   - **Routing key**: `agent.chat.request`
   - **Properties**: `reply_to`, `correlation_id`, `content_type = application/json`
   - **Формат тела сообщения**:
     ```json
     {
       "request_id": "req_1741508400000_1_a1b2c3d4",
       "action": "chat",
       "payload": {
         "user_id": "15",
         "session_id": "1",
         "deal_id": null,
         "message": "Привет! Расскажи, чем ты можешь помочь менеджеру?"
       }
     }
     ```
3. `Agent Service` обрабатывает запрос через GigaChat и публикует ответ в очередь `reply_to`:
   ```json
   {
     "request_id": "req_1741508400000_1_a1b2c3d4",
     "success": true,
     "data": {
       "message": "Привет! Я могу подобрать альтернативные планировки по параметрам клиента...",
       "agent": "general",
       "intent": "general_chat"
     },
     "error": null
   }
   ```
4. `Go Backend` сверяет `correlation_id` и `request_id`, сохраняет ответ в базу (если указан `session_id`) и возвращает JSON клиенту:
   ```json
   {
     "message": "Привет! Я могу подобрать альтернативные планировки по параметрам клиента...",
     "agent": "general",
     "intent": "general_chat"
   }
   ```
