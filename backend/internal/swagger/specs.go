package swagger

var UserSpec = []byte(`{
  "openapi": "3.0.3",
  "info": {
    "title": "AI Hackathon - User Server API",
    "description": "Клиентский API для покупателей недвижимости, взаимодействия с менеджерами, просмотра сделок и предложений",
    "version": "1.0.0"
  },
  "servers": [
    {
      "url": "http://localhost:8080",
      "description": "Local User Server"
    }
  ],
  "components": {
    "securitySchemes": {
      "BearerAuth": {
        "type": "http",
        "scheme": "bearer",
        "bearerFormat": "JWT",
        "description": "Введите JWT токен, полученный при логине"
      }
    },
    "schemas": {
      "RegisterRequest": {
        "type": "object",
        "required": ["name", "email", "password"],
        "properties": {
          "name": { "type": "string", "example": "Иван Иванов" },
          "email": { "type": "string", "example": "ivan@example.com" },
          "password": { "type": "string", "example": "secret123" }
        }
      },
      "LoginRequest": {
        "type": "object",
        "required": ["email", "password"],
        "properties": {
          "email": { "type": "string", "example": "ivan@example.com" },
          "password": { "type": "string", "example": "secret123" }
        }
      },
      "UpdateUserRequest": {
        "type": "object",
        "properties": {
          "name": { "type": "string", "example": "Иван Петров" },
          "email": { "type": "string", "example": "ivan.petrov@example.com" },
          "password": { "type": "string", "example": "newpassword123" }
        }
      },
      "CreateChatSessionRequest": {
        "type": "object",
        "properties": {
          "id_apartment": { "type": "integer", "example": 42 },
          "guest_name": { "type": "string", "example": "Иван" },
          "guest_email": { "type": "string", "example": "ivan@example.com" },
          "guest_phone": { "type": "string", "example": "+79991234567" },
          "message": { "type": "string", "example": "Здравствуйте! Хочу узнать условия бронирования." }
        }
      },
      "SendMessageRequest": {
        "type": "object",
        "required": ["content"],
        "properties": {
          "content": { "type": "string", "example": "Возможна ли рассрочка на 1 год?" }
        }
      },
      "AIChatRequest": {
        "type": "object",
        "required": ["message", "session_id"],
        "properties": {
          "message": { "type": "string", "example": "Привет! Расскажи, чем ты можешь помочь?" },
          "session_id": { "type": "integer", "example": 1 },
          "deal_id": { "type": "integer", "nullable": true, "example": null }
        }
      },
      "AIChatResponse": {
        "type": "object",
        "properties": {
          "message": { "type": "string", "example": "Здравствуйте! Я могу подобрать вам квартиру..." },
          "agent": { "type": "string", "example": "general" },
          "intent": { "type": "string", "example": "general_chat" }
        }
      },
      "ChatSession": {
        "type": "object",
        "properties": {
          "id": { "type": "integer", "example": 1 },
          "id_user": { "type": "integer", "nullable": true, "example": 10 },
          "id_employee": { "type": "integer", "nullable": true, "example": 99 },
          "id_apartment": { "type": "integer", "nullable": true, "example": 42 },
          "guest_name": { "type": "string", "example": "Иван" },
          "guest_email": { "type": "string", "example": "ivan@example.com" },
          "guest_phone": { "type": "string", "example": "+79991234567" },
          "status": { "type": "string", "enum": ["open", "in_progress", "close"], "example": "open" },
          "created_at": { "type": "string", "format": "date-time" },
          "updated_at": { "type": "string", "format": "date-time" }
        }
      },
      "Message": {
        "type": "object",
        "properties": {
          "id": { "type": "integer", "example": 1 },
          "id_chat_session": { "type": "integer", "example": 1 },
          "id_user": { "type": "integer", "nullable": true, "example": 10 },
          "sender_type": { "type": "string", "enum": ["client", "manager", "ai", "system"], "example": "client" },
          "content": { "type": "string", "example": "Здравствуйте!" },
          "is_read": { "type": "boolean", "example": false },
          "sended_at": { "type": "string", "format": "date-time" }
        }
      },
      "Deal": {
        "type": "object",
        "properties": {
          "id": { "type": "integer", "example": 1 },
          "id_user": { "type": "integer", "example": 10 },
          "id_employee": { "type": "integer", "example": 99 },
          "id_apartment": { "type": "integer", "example": 42 },
          "base_price": { "type": "number", "example": 10000000.00 },
          "percent_discount": { "type": "number", "example": 5.0 },
          "total_price": { "type": "number", "example": 9500000.00 },
          "status": { "type": "string", "enum": ["pending", "contract", "completed", "cancelled"], "example": "pending" },
          "created_at": { "type": "string", "format": "date-time" },
          "updated_at": { "type": "string", "format": "date-time" }
        }
      }
    }
  },
  "paths": {
    "/api/auth/register": {
      "post": {
        "tags": ["Auth"],
        "summary": "Регистрация нового клиента",
        "requestBody": {
          "required": true,
          "content": { "application/json": { "schema": { "$ref": "#/components/schemas/RegisterRequest" } } }
        },
        "responses": {
          "201": { "description": "Пользователь успешно зарегистрирован" },
          "409": { "description": "Пользователь с таким email уже существует" }
        }
      }
    },
    "/api/auth/login": {
      "post": {
        "tags": ["Auth"],
        "summary": "Вход клиента (JWT токен и Cookie)",
        "requestBody": {
          "required": true,
          "content": { "application/json": { "schema": { "$ref": "#/components/schemas/LoginRequest" } } }
        },
        "responses": {
          "200": { "description": "Успешный вход" },
          "401": { "description": "Неверные учетные данные" }
        }
      }
    },
    "/api/auth/logout": {
      "post": {
        "tags": ["Auth"],
        "summary": "Выход из системы",
        "responses": {
          "200": { "description": "Успешный выход" }
        }
      }
    },
    "/api/auth/profile": {
      "get": {
        "tags": ["Auth"],
        "summary": "Получить профиль текущего пользователя",
        "security": [{ "BearerAuth": [] }],
        "responses": {
          "200": { "description": "Данные профиля" },
          "401": { "description": "Неавторизован" }
        }
      }
    },
    "/api/users/me": {
      "get": {
        "tags": ["Users"],
        "summary": "Получить данные текущего пользователя",
        "security": [{ "BearerAuth": [] }],
        "responses": {
          "200": { "description": "Данные пользователя" }
        }
      },
      "put": {
        "tags": ["Users"],
        "summary": "Обновить данные текущего пользователя",
        "security": [{ "BearerAuth": [] }],
        "requestBody": {
          "required": true,
          "content": { "application/json": { "schema": { "$ref": "#/components/schemas/UpdateUserRequest" } } }
        },
        "responses": {
          "200": { "description": "Данные успешно обновлены" }
        }
      }
    },
    "/api/complexes": {
      "get": {
        "tags": ["Catalog"],
        "summary": "Список всех жилых комплексов",
        "responses": { "200": { "description": "Список ЖК" } }
      }
    },
    "/api/complexes/{id}": {
      "get": {
        "tags": ["Catalog"],
        "summary": "Детали конкретного ЖК",
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "integer" } }],
        "responses": { "200": { "description": "Данные ЖК" }, "404": { "description": "ЖК не найден" } }
      }
    },
    "/api/complexes/{complexId}/buildings": {
      "get": {
        "tags": ["Catalog"],
        "summary": "Корпуса конкретного ЖК",
        "parameters": [{ "name": "complexId", "in": "path", "required": true, "schema": { "type": "integer" } }],
        "responses": { "200": { "description": "Список корпусов" } }
      }
    },
    "/api/buildings/{id}": {
      "get": {
        "tags": ["Catalog"],
        "summary": "Детали корпуса здания",
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "integer" } }],
        "responses": { "200": { "description": "Данные корпуса" } }
      }
    },
    "/api/buildings/{buildingId}/apartments": {
      "get": {
        "tags": ["Catalog"],
        "summary": "Список квартир в корпусе",
        "parameters": [{ "name": "buildingId", "in": "path", "required": true, "schema": { "type": "integer" } }],
        "responses": { "200": { "description": "Список квартир" } }
      }
    },
    "/api/buildings/{buildingId}/progress": {
      "get": {
        "tags": ["Catalog"],
        "summary": "Ход строительства корпуса",
        "parameters": [{ "name": "buildingId", "in": "path", "required": true, "schema": { "type": "integer" } }],
        "responses": { "200": { "description": "Этапы строительства" } }
      }
    },
    "/api/apartments/{id}": {
      "get": {
        "tags": ["Catalog"],
        "summary": "Детали квартиры",
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "integer" } }],
        "responses": { "200": { "description": "Данные квартиры" } }
      }
    },
    "/api/chat/sessions": {
      "post": {
        "tags": ["Chat"],
        "summary": "Создать сессию чата / заявку (авторизованный клиент или гость)",
        "requestBody": {
          "required": true,
          "content": { "application/json": { "schema": { "$ref": "#/components/schemas/CreateChatSessionRequest" } } }
        },
        "responses": {
          "201": { "description": "Сессия успешно создана", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ChatSession" } } } }
        }
      },
      "get": {
        "tags": ["Chat"],
        "summary": "Список своих чат-сессий",
        "security": [{ "BearerAuth": [] }],
        "responses": {
          "200": { "description": "Список сессий", "content": { "application/json": { "schema": { "type": "array", "items": { "$ref": "#/components/schemas/ChatSession" } } } } }
        }
      }
    },
    "/api/chat/sessions/{id}": {
      "get": {
        "tags": ["Chat"],
        "summary": "Получить детали сессии",
        "security": [{ "BearerAuth": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "integer" } }],
        "responses": {
          "200": { "description": "Детали сессии", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ChatSession" } } } }
        }
      }
    },
    "/api/chat/sessions/{id}/messages": {
      "get": {
        "tags": ["Chat"],
        "summary": "Получить сообщения сессии (помечает прочитанными)",
        "security": [{ "BearerAuth": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "integer" } }],
        "responses": {
          "200": { "description": "Список сообщений", "content": { "application/json": { "schema": { "type": "array", "items": { "$ref": "#/components/schemas/Message" } } } } }
        }
      },
      "post": {
        "tags": ["Chat"],
        "summary": "Отправить сообщение в сессию",
        "security": [{ "BearerAuth": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "integer" } }],
        "requestBody": {
          "required": true,
          "content": { "application/json": { "schema": { "$ref": "#/components/schemas/SendMessageRequest" } } }
        },
        "responses": {
          "201": { "description": "Сообщение отправлено", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/Message" } } } }
        }
      }
    },
    "/api/deals": {
      "get": {
        "tags": ["Deals"],
        "summary": "Список предложений и сделок клиента",
        "security": [{ "BearerAuth": [] }],
        "responses": {
          "200": { "description": "Список сделок", "content": { "application/json": { "schema": { "type": "array", "items": { "$ref": "#/components/schemas/Deal" } } } } }
        }
      }
    },
    "/api/deals/{id}": {
      "get": {
        "tags": ["Deals"],
        "summary": "Детали сделки",
        "security": [{ "BearerAuth": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "integer" } }],
        "responses": {
          "200": { "description": "Детали сделки", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/Deal" } } } }
        }
      }
    },
    "/api/ai/chat": {
      "post": {
        "tags": ["AI"],
        "summary": "Чат с AI Agent (GigaChat) через RabbitMQ RPC",
        "security": [{ "BearerAuth": [] }],
        "requestBody": {
          "required": true,
          "content": { "application/json": { "schema": { "$ref": "#/components/schemas/AIChatRequest" } } }
        },
        "responses": {
          "200": { "description": "Ответ от Agent Service", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/AIChatResponse" } } } },
          "503": { "description": "RabbitMQ / AI сервис временно недоступен" },
          "504": { "description": "Таймаут ожидания ответа от AI" }
        }
      }
    }
  }
}`)

var StaffSpec = []byte(`{
  "openapi": "3.0.3",
  "info": {
    "title": "AI Hackathon - Staff Server API",
    "description": "Корпоративный API для менеджеров и супервайзеров (управление заявками, сделками, объектами недвижимости и AI ассистентом)",
    "version": "1.0.0"
  },
  "servers": [
    {
      "url": "http://localhost:8081",
      "description": "Local Staff Server"
    }
  ],
  "components": {
    "securitySchemes": {
      "BearerAuth": {
        "type": "http",
        "scheme": "bearer",
        "bearerFormat": "JWT",
        "description": "JWT токен сотрудника (роль manager или supervisor)"
      }
    },
    "schemas": {
      "CreateDealRequest": {
        "type": "object",
        "required": ["id_user", "id_apartment", "base_price", "percent_discount"],
        "properties": {
          "id_user": { "type": "integer", "example": 10 },
          "id_apartment": { "type": "integer", "example": 42 },
          "base_price": { "type": "number", "example": 10000000.00 },
          "percent_discount": { "type": "number", "example": 5.0 }
        }
      },
      "UpdateDealStatusRequest": {
        "type": "object",
        "required": ["status"],
        "properties": {
          "status": { "type": "string", "enum": ["pending", "contract", "completed", "cancelled"], "example": "contract" },
          "percent_discount": { "type": "number", "example": 7.0 }
        }
      },
      "RejectSessionRequest": {
        "type": "object",
        "required": ["reason"],
        "properties": {
          "reason": { "type": "string", "example": "Клиент выбрал другой ЖК, не устроила цена" }
        }
      },
      "AIChatRequest": {
        "type": "object",
        "required": ["message", "session_id"],
        "properties": {
          "message": { "type": "string", "example": "Предложи клиенту альтернативные 2-комнатные квартиры до 12 млн" },
          "session_id": { "type": "integer", "example": 1 },
          "deal_id": { "type": "integer", "nullable": true, "example": null }
        }
      }
    }
  },
  "paths": {
    "/api/auth/register": {
      "post": {
        "tags": ["Auth"],
        "summary": "Создание аккаунта сотрудника (manager/supervisor)",
        "requestBody": {
          "required": true,
          "content": { "application/json": { "schema": { "type": "object", "required": ["name", "email", "password", "role"], "properties": { "name": { "type": "string" }, "email": { "type": "string" }, "password": { "type": "string" }, "role": { "type": "string", "enum": ["manager", "supervisor"] } } } } }
        },
        "responses": { "201": { "description": "Сотрудник создан" } }
      }
    },
    "/api/auth/login": {
      "post": {
        "tags": ["Auth"],
        "summary": "Вход сотрудника",
        "requestBody": {
          "required": true,
          "content": { "application/json": { "schema": { "type": "object", "required": ["email", "password"], "properties": { "email": { "type": "string" }, "password": { "type": "string" } } } } }
        },
        "responses": { "200": { "description": "Успешный вход" } }
      }
    },
    "/api/staff/profile": {
      "get": {
        "tags": ["Staff"],
        "summary": "Профиль текущего сотрудника",
        "security": [{ "BearerAuth": [] }],
        "responses": { "200": { "description": "Профиль сотрудника" } }
      }
    },
    "/api/users": {
      "get": {
        "tags": ["Users"],
        "summary": "Список всех зарегистрированных пользователей",
        "security": [{ "BearerAuth": [] }],
        "responses": { "200": { "description": "Список пользователей" } }
      }
    },
    "/api/chat/sessions": {
      "get": {
        "tags": ["Chat Management"],
        "summary": "Список всех обращений и заявок с фильтрацией",
        "security": [{ "BearerAuth": [] }],
        "parameters": [
          { "name": "status", "in": "query", "schema": { "type": "string", "enum": ["open", "in_progress", "close"] }, "description": "Фильтр по статусу" },
          { "name": "employee_id", "in": "query", "schema": { "type": "integer" }, "description": "Фильтр по назначенному сотруднику" }
        ],
        "responses": { "200": { "description": "Список сессий" } }
      }
    },
    "/api/chat/sessions/{id}": {
      "get": {
        "tags": ["Chat Management"],
        "summary": "Детали сессии и контакты клиента",
        "security": [{ "BearerAuth": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "integer" } }],
        "responses": { "200": { "description": "Детали сессии" } }
      }
    },
    "/api/chat/sessions/{id}/take": {
      "post": {
        "tags": ["Chat Management"],
        "summary": "Взять заявку в работу (назначает текущего сотрудника, статус -> in_progress)",
        "security": [{ "BearerAuth": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "integer" } }],
        "responses": { "200": { "description": "Заявка переведена в работу" } }
      }
    },
    "/api/chat/sessions/{id}/close": {
      "post": {
        "tags": ["Chat Management"],
        "summary": "Завершить сессию чата (статус -> close)",
        "security": [{ "BearerAuth": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "integer" } }],
        "responses": { "200": { "description": "Сессия закрыта" } }
      }
    },
    "/api/chat/sessions/{id}/reject": {
      "post": {
        "tags": ["Chat Management"],
        "summary": "Отклонить заявку с указанием причины (сохраняется в chat_session_rejections)",
        "security": [{ "BearerAuth": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "integer" } }],
        "requestBody": {
          "required": true,
          "content": { "application/json": { "schema": { "$ref": "#/components/schemas/RejectSessionRequest" } } }
        },
        "responses": { "201": { "description": "Отказ зафиксирован" } }
      }
    },
    "/api/chat/sessions/{id}/messages": {
      "get": {
        "tags": ["Chat Management"],
        "summary": "Получить сообщения сессии",
        "security": [{ "BearerAuth": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "integer" } }],
        "responses": { "200": { "description": "Список сообщений" } }
      },
      "post": {
        "tags": ["Chat Management"],
        "summary": "Отправить ответ клиенту",
        "security": [{ "BearerAuth": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "integer" } }],
        "requestBody": {
          "required": true,
          "content": { "application/json": { "schema": { "type": "object", "required": ["content"], "properties": { "content": { "type": "string" } } } } }
        },
        "responses": { "201": { "description": "Сообщение отправлено" } }
      }
    },
    "/api/deals": {
      "post": {
        "tags": ["Deals Management"],
        "summary": "Создать сделку / коммерческое предложение",
        "security": [{ "BearerAuth": [] }],
        "requestBody": {
          "required": true,
          "content": { "application/json": { "schema": { "$ref": "#/components/schemas/CreateDealRequest" } } }
        },
        "responses": { "201": { "description": "Сделка создана" } }
      },
      "get": {
        "tags": ["Deals Management"],
        "summary": "Список всех сделок (фильтрация по status, user_id)",
        "security": [{ "BearerAuth": [] }],
        "parameters": [
          { "name": "status", "in": "query", "schema": { "type": "string", "enum": ["pending", "contract", "completed", "cancelled"] } },
          { "name": "user_id", "in": "query", "schema": { "type": "integer" } }
        ],
        "responses": { "200": { "description": "Список сделок" } }
      }
    },
    "/api/deals/{id}": {
      "get": {
        "tags": ["Deals Management"],
        "summary": "Детали сделки",
        "security": [{ "BearerAuth": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "integer" } }],
        "responses": { "200": { "description": "Детали сделки" } }
      }
    },
    "/api/deals/{id}/status": {
      "put": {
        "tags": ["Deals Management"],
        "summary": "Обновить статус сделки и/или процент скидки",
        "security": [{ "BearerAuth": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "integer" } }],
        "requestBody": {
          "required": true,
          "content": { "application/json": { "schema": { "$ref": "#/components/schemas/UpdateDealStatusRequest" } } }
        },
        "responses": { "200": { "description": "Сделка обновлена" } }
      }
    },
    "/api/ai/chat": {
      "post": {
        "tags": ["AI Assistant"],
        "summary": "Консультация сотрудника с AI Agent (подбор альтернатив, КП)",
        "security": [{ "BearerAuth": [] }],
        "requestBody": {
          "required": true,
          "content": { "application/json": { "schema": { "$ref": "#/components/schemas/AIChatRequest" } } }
        },
        "responses": { "200": { "description": "Ответ AI Agent" } }
      }
    }
  }
}`)
