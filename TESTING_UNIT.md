# Руководство по Unit тестированию

## Запуск тестов

### Запуск всех тестов

```bash
# Запустить все тесты
make test

# Или напрямую через docker
docker run --rm -v $(pwd):/app -w /app golang:1.22-alpine go test -v ./...
```

### Запуск конкретных тестов

```bash
# Только тесты auth handler
docker run --rm -v $(pwd):/app -w /app golang:1.22-alpine \
  go test -v ./internal/delivery/http -run TestAuthHandler

# Только тесты vehicle handler
docker run --rm -v $(pwd):/app -w /app golang:1.22-alpine \
  go test -v ./internal/delivery/http -run TestVehicleHandler

# Конкретный тест
docker run --rm -v $(pwd):/app -w /app golang:1.22-alpine \
  go test -v ./internal/delivery/http -run TestAuthHandler_Login
```

### Запуск тестов с покрытием

```bash
# Генерация отчета о покрытии
docker run --rm -v $(pwd):/app -w /app golang:1.22-alpine \
  go test -coverprofile=coverage.out ./internal/delivery/http

# Просмотр покрытия в консоли
docker run --rm -v $(pwd):/app -w /app golang:1.22-alpine \
  go tool cover -func=coverage.out

# Генерация HTML отчета
docker run --rm -v $(pwd):/app -w /app golang:1.22-alpine \
  go tool cover -html=coverage.out -o coverage.html
```

## Структура тестов

### Auth Handler Tests (`auth_handler_test.go`)

**Покрытие:**
- ✅ `POST /api/v1/auth/register` - Регистрация пользователя
  - Успешная регистрация
  - Пользователь уже существует (409 Conflict)
  - Невалидный JSON (400 Bad Request)

- ✅ `POST /api/v1/auth/login` - Вход пользователя
  - Успешный вход
  - Неверные учетные данные (401 Unauthorized)
  - Неактивный пользователь (403 Forbidden)

- ✅ `POST /api/v1/auth/logout` - Выход пользователя
  - Успешный выход
  - Невалидный refresh token (401 Unauthorized)

- ✅ `POST /api/v1/auth/refresh` - Обновление токена
  - Успешное обновление
  - Невалидный refresh token (401 Unauthorized)
  - Пользователь не найден (401 Unauthorized)

### Vehicle Handler Tests (`vehicle_handler_test.go`)

**Покрытие:**
- ✅ `POST /api/v1/vehicles` - Создание автомобиля
  - Успешное создание
  - Дублирующийся номер (409 Conflict)
  - Невалидный JSON (400 Bad Request)

- ✅ `GET /api/v1/vehicles/me` - Получение моих автомобилей
  - Успешное получение списка
  - Пустой список автомобилей

- ✅ `GET /api/v1/vehicles/:id` - Получение автомобиля по ID
  - Успешное получение
  - Автомобиль не найден (404 Not Found)
  - Невалидный UUID (400 Bad Request)

### Pass Handler Tests (`pass_handler_test.go`)

**Покрытие:**
- ✅ `POST /api/v1/passes` - Создание пропуска
  - Успешное создание
  - Отсутствие авторизации (401 Unauthorized)
  - Невалидный JSON (400 Bad Request)

- ✅ `GET /api/v1/passes/me` - Получение моих пропусков
  - Успешное получение списка
  - Пустой список пропусков
  - Отсутствие авторизации (401 Unauthorized)

- ✅ `GET /api/v1/passes/:id` - Получение пропуска по ID
  - Успешное получение
  - Пропуск не найден (404 Not Found)
  - Невалидный UUID (400 Bad Request)

- ✅ `DELETE /api/v1/passes/:id/revoke` - Отзыв пропуска
  - Успешный отзыв
  - Пропуск не найден (404 Not Found)
  - Невалидный UUID (400 Bad Request)
  - Отсутствие авторизации (401 Unauthorized)

### Access Handler Tests (`access_handler_test.go`)

**Покрытие:**
- ✅ `POST /api/v1/access/check` - Проверка доступа
  - Успешная проверка - доступ разрешен
  - Успешная проверка - доступ запрещен
  - Невалидный JSON (400 Bad Request)

- ✅ `GET /api/v1/access/logs` - Получение всех логов
  - Успешное получение без фильтра
  - Получение с пагинацией (limit, offset)
  - Фильтрация по user_id
  - Невалидный user_id (400 Bad Request)

- ✅ `GET /api/v1/access/logs/vehicle/:id` - Логи автомобиля
  - Успешное получение логов
  - Невалидный vehicle ID (400 Bad Request)
  - Пустая история проездов

- ✅ `GET /api/v1/access/me/logs` - Мои логи проездов
  - Успешное получение моих логов
  - Отсутствие авторизации (401 Unauthorized)
  - Пустая история проездов

## Моки (Mocks)

Все тесты используют моки для сервисов:

- **MockAuthService** - мок для `auth.Service`
  - Register, Login, Logout, RefreshToken, GetUserByID
- **MockVehicleService** - мок для `vehicle.Service`
  - CreateVehicle, GetVehiclesByOwner, GetVehicleByID
- **MockPassService** - мок для `pass.Service`
  - CreatePass, GetPassesByUser, GetPassByID, RevokePass
- **MockAccessService** - мок для `access.Service`
  - CheckAccess, GetAccessLogs, GetAccessLogsByVehicle

Моки создаются с помощью библиотеки `testify/mock`.

## Тестовые утилиты (`test_helpers.go`)

**Вспомогательные функции:**
- `CreateTestUser()` - создание тестового пользователя
- `CreateTestVehicle()` - создание тестового автомобиля
- `CreateTestPass()` - создание тестового пропуска
- `CreateTestAccessLog()` - создание тестового лога доступа
- `CreateAuthContext()` - создание контекста с user_id
- `CreateTestJWTToken()` - генерация тестового JWT токена
- `AssertSuccess()` - проверка успешного ответа API
- `AssertError()` - проверка ошибочного ответа API

## Зависимости для тестирования

Тесты используют следующие библиотеки:

```go
github.com/stretchr/testify/assert   // Ассерты
github.com/stretchr/testify/mock     // Моки
github.com/go-chi/chi/v5             // Router для тестирования параметров
```

Установка зависимостей:

```bash
go get github.com/stretchr/testify/assert
go get github.com/stretchr/testify/mock
```

## Структура теста

```go
func TestHandlerName_MethodName(t *testing.T) {
    tests := []struct {
        name           string              // Название теста
        requestBody    interface{}         // Тело запроса
        mockSetup      func(*MockService)  // Настройка мока
        expectedStatus int                 // Ожидаемый статус
        checkResponse  func(*testing.T, map[string]interface{}) // Проверка ответа
    }{
        {
            name: "успешный сценарий",
            requestBody: RequestStruct{...},
            mockSetup: func(m *MockService) {
                m.On("Method", mock.Anything, mock.Anything).Return(result, nil)
            },
            expectedStatus: http.StatusOK,
            checkResponse: func(t *testing.T, resp map[string]interface{}) {
                assert.True(t, resp["success"].(bool))
                // Дополнительные проверки...
            },
        },
        // Дополнительные сценарии...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Настройка мока
            mockService := new(MockService)
            tt.mockSetup(mockService)

            // Создание handler
            handler := NewHandler(mockService, logger)

            // Создание HTTP запроса
            body, _ := json.Marshal(tt.requestBody)
            req := httptest.NewRequest(http.MethodPost, "/path", bytes.NewReader(body))
            w := httptest.NewRecorder()

            // Выполнение запроса
            handler.Method(w, req)

            // Проверки
            assert.Equal(t, tt.expectedStatus, w.Code)
            var response map[string]interface{}
            json.Unmarshal(w.Body.Bytes(), &response)
            tt.checkResponse(t, response)

            mockService.AssertExpectations(t)
        })
    }
}
```

## Best Practices

1. **Используй table-driven tests** - один тест функция, много тест кейсов
2. **Проверяй все пути выполнения** - success, error, validation
3. **Мокируй зависимости** - не используй реальную БД в unit тестах
4. **Проверяй HTTP статусы** - 200, 400, 401, 403, 404, 409, 500
5. **Проверяй структуру ответа** - success flag, data/error поля
6. **Изолируй тесты** - каждый тест независим и может выполняться отдельно

## Пример запуска

```bash
# 1. Запустить все тесты
make test

# 2. Проверить результат
=== RUN   TestAuthHandler_Register
=== RUN   TestAuthHandler_Register/успешная_регистрация
=== RUN   TestAuthHandler_Register/пользователь_уже_существует
=== RUN   TestAuthHandler_Register/невалидный_JSON
--- PASS: TestAuthHandler_Register (0.01s)
    --- PASS: TestAuthHandler_Register/успешная_регистрация (0.00s)
    --- PASS: TestAuthHandler_Register/пользователь_уже_существует (0.00s)
    --- PASS: TestAuthHandler_Register/невалидный_JSON (0.00s)
PASS
ok      github.com/frontandrew/gate/internal/delivery/http     0.123s
```

## TODO: Дополнительные тесты

Следующие тесты нужно добавить:

- [x] Pass Handler tests (`pass_handler_test.go`) ✅
  - GET /api/v1/passes/me
  - GET /api/v1/passes/:id
  - POST /api/v1/passes (admin/guard)
  - DELETE /api/v1/passes/:id/revoke (admin/guard)

- [x] Access Handler tests (`access_handler_test.go`) ✅
  - POST /api/v1/access/check
  - GET /api/v1/access/me/logs
  - GET /api/v1/access/logs/vehicle/:id
  - GET /api/v1/access/logs (admin/guard)

- [ ] Middleware tests
  - AuthMiddleware
  - RequireRole middleware
  - CORS middleware
  - Logging middleware

- [ ] Integration tests
  - Полный flow: register → login → create vehicle → create pass → check access
  - С реальной тестовой БД (testcontainers)

---

## Текущий статус

**Результаты тестов: 28/28 scenarios PASS ✅**

```bash
$ make test
✅ TestAuthHandler_Register - 3/3 scenarios
✅ TestAuthHandler_Login - 3/3 scenarios
✅ TestAuthHandler_Logout - 2/2 scenarios
✅ TestAuthHandler_RefreshToken - 3/3 scenarios
✅ TestPassHandler_CreatePass - 3/3 scenarios
✅ TestPassHandler_GetMyPasses - 3/3 scenarios
✅ TestPassHandler_GetPassByID - 3/3 scenarios
✅ TestPassHandler_RevokePass - 4/4 scenarios
✅ TestVehicleHandler_CreateVehicle - 3/3 scenarios
✅ TestVehicleHandler_GetMyVehicles - 2/2 scenarios
✅ TestVehicleHandler_GetVehicleByID - 3/3 scenarios

PASS - ok  github.com/frontandrew/gate/internal/delivery/http
```

**Покрытие:** 11 endpoints, ~75% HTTP handlers (auth + vehicle + pass)
**Цель:** 80%+ покрытие для production готовности - **ПОЧТИ ДОСТИГНУТО**
