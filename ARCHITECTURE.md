# Архитектура системы GATE - Обновленная версия

## Ключевые архитектурные принципы

### Центральная сущность: Пользователь

**Важно**: Автомобильный номер - это способ аутентификации пользователя в системе, а не самостоятельная сущность.

**Правильная логика доступа:**
```
Номер авто → Автомобиль → Владелец (Пользователь) → Активный пропуск → Решение о доступе
```

**Ключевые факты:**
- ✅ Автомобиль не может существовать без владельца (owner_id NOT NULL)
- ✅ Пропуск выдается пользователю, а не автомобилю
- ✅ В логах фиксируется, какой пользователь получил доступ
- ✅ Статистика строится по пользователям, а не по автомобилям

## Обновленная схема базы данных

### Итерация 1: MVP - все основные таблицы

```sql
-- 1. USERS - центральная таблица (ПЕРВАЯ!)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    phone VARCHAR(20),
    role VARCHAR(20) NOT NULL DEFAULT 'user',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT email_format CHECK (email ~ '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    CONSTRAINT role_check CHECK (role IN ('admin', 'user', 'guard'))
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);

-- 2. VEHICLES - ВСЕГДА привязаны к пользователю
CREATE TABLE vehicles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,  -- ОБЯЗАТЕЛЬНАЯ связь
    license_plate VARCHAR(20) NOT NULL UNIQUE,
    vehicle_type vehicle_type_enum DEFAULT 'car',
    model VARCHAR(100),
    color VARCHAR(50),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT license_plate_format CHECK (license_plate ~ '^[A-ZА-Я0-9]+$')
);

CREATE INDEX idx_vehicles_owner_id ON vehicles(owner_id);
CREATE INDEX idx_vehicles_license_plate ON vehicles(license_plate);
CREATE INDEX idx_vehicles_is_active ON vehicles(is_active) WHERE is_active = true;

-- 3. PASSES - пропуска выдаются ПОЛЬЗОВАТЕЛЮ
CREATE TABLE passes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    pass_type VARCHAR(20) NOT NULL DEFAULT 'permanent',
    valid_from TIMESTAMP NOT NULL DEFAULT NOW(),
    valid_until TIMESTAMP,
    is_active BOOLEAN NOT NULL DEFAULT true,
    revoked_at TIMESTAMP,
    revoked_by UUID REFERENCES users(id),
    revoke_reason VARCHAR(500),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT pass_type_check CHECK (pass_type IN ('permanent', 'temporary')),
    CONSTRAINT valid_dates_check CHECK (valid_until IS NULL OR valid_until > valid_from),
    -- Один пользователь может иметь только один активный пропуск
    CONSTRAINT unique_active_pass UNIQUE (user_id, is_active)
);

CREATE INDEX idx_passes_user_id ON passes(user_id);
CREATE INDEX idx_passes_valid_dates ON passes(valid_from, valid_until);
CREATE INDEX idx_passes_is_active ON passes(is_active) WHERE is_active = true;

-- 4. ACCESS_LOGS - фиксируют пользователя и способ доступа
CREATE TABLE access_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),           -- КТО получил доступ (главное!)
    vehicle_id UUID NOT NULL REFERENCES vehicles(id),     -- КАКИМ способом (через какой номер)
    license_plate VARCHAR(20) NOT NULL,                   -- Распознанный номер
    image_url VARCHAR(500),                               -- Фото с камеры
    recognition_confidence DECIMAL(5,2) CHECK (recognition_confidence >= 0 AND recognition_confidence <= 100),
    access_granted BOOLEAN NOT NULL,
    access_reason VARCHAR(255),                           -- Причина решения
    gate_id VARCHAR(50),
    direction direction_enum NOT NULL DEFAULT 'IN',
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT direction_check CHECK (direction IN ('IN', 'OUT'))
);

CREATE INDEX idx_access_logs_user_id ON access_logs(user_id);
CREATE INDEX idx_access_logs_vehicle_id ON access_logs(vehicle_id);
CREATE INDEX idx_access_logs_timestamp ON access_logs(timestamp DESC);
CREATE INDEX idx_access_logs_license_plate ON access_logs(license_plate);
CREATE INDEX idx_access_logs_gate_id ON access_logs(gate_id);
CREATE INDEX idx_access_logs_access_granted ON access_logs(access_granted);

-- 5. REFRESH_TOKENS - для JWT аутентификации
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMP
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);

-- Комментарии
COMMENT ON TABLE users IS 'Пользователи системы - центральная сущность';
COMMENT ON TABLE vehicles IS 'Автомобили пользователей - способ аутентификации';
COMMENT ON TABLE passes IS 'Пропуска выдаются пользователям, а не автомобилям';
COMMENT ON TABLE access_logs IS 'История доступа: кто (user), через что (vehicle), когда';

COMMENT ON COLUMN access_logs.user_id IS 'Главная информация - какой пользователь получил доступ';
COMMENT ON COLUMN access_logs.vehicle_id IS 'Вспомогательная информация - через какой автомобиль';
```

## Обновленная логика проверки доступа

### Схема проверки

```
┌─────────────────────────────────────────────────────────────────┐
│                    Процесс проверки доступа                     │
└─────────────────────────────────────────────────────────────────┘

1. Камера фиксирует номер автомобиля
                ↓
2. ML сервис распознает номер → "А123ВС777"
                ↓
3. Поиск автомобиля в БД по номеру
                ↓
4. Получение владельца автомобиля (User)  ← КЛЮЧЕВОЙ МОМЕНТ!
                ↓
5. Проверка активного пропуска пользователя
                ↓
6. Проверка временных ограничений (если пропуск временный)
                ↓
7. Запись в access_logs (user_id + vehicle_id)
                ↓
8. Решение: доступ разрешен/запрещен
```

### Код проверки доступа (Go)

```go
// internal/usecase/access/service.go

type AccessService struct {
    userRepo    repository.UserRepository
    vehicleRepo repository.VehicleRepository
    passRepo    repository.PassRepository
    logRepo     repository.AccessLogRepository
    mlClient    ml.MLClient
    logger      logger.Logger
}

func (s *AccessService) CheckAccess(ctx context.Context, req *CheckAccessRequest) (*CheckAccessResponse, error) {
    // 1. Распознать номер через ML сервис
    recognized, err := s.mlClient.RecognizePlate(ctx, req.Image)
    if err != nil {
        return nil, fmt.Errorf("failed to recognize plate: %w", err)
    }

    // 2. Найти автомобиль по номеру
    vehicle, err := s.vehicleRepo.GetByLicensePlate(ctx, recognized.LicensePlate)
    if err != nil {
        return s.denyAccess(ctx, recognized.LicensePlate, "Vehicle not found", req)
    }

    // 3. КЛЮЧЕВОЕ: получить владельца автомобиля
    user, err := s.userRepo.GetByID(ctx, vehicle.OwnerID)
    if err != nil {
        return s.denyAccess(ctx, recognized.LicensePlate, "Owner not found", req)
    }

    // Проверить, что пользователь активен
    if !user.IsActive {
        return s.denyAccess(ctx, recognized.LicensePlate, "User is not active", req)
    }

    // 4. Проверить активный пропуск ПОЛЬЗОВАТЕЛЯ
    pass, err := s.passRepo.GetActivePassByUser(ctx, user.ID)
    if err != nil || pass == nil {
        return s.denyAccess(ctx, recognized.LicensePlate, "No valid pass for user", req)
    }

    // 5. Проверить временные ограничения
    if pass.PassType == "temporary" {
        now := time.Now()
        if now.Before(pass.ValidFrom) {
            return s.denyAccess(ctx, recognized.LicensePlate, "Pass not yet valid", req)
        }
        if pass.ValidUntil != nil && now.After(*pass.ValidUntil) {
            return s.denyAccess(ctx, recognized.LicensePlate, "Pass expired", req)
        }
    }

    // 6. Записать лог успешного доступа
    accessLog := &domain.AccessLog{
        UserID:                 user.ID,              // ГЛАВНОЕ: кто получил доступ
        VehicleID:              vehicle.ID,           // Через какой транспорт
        LicensePlate:           recognized.LicensePlate,
        ImageURL:               req.ImageURL,
        RecognitionConfidence:  recognized.Confidence,
        AccessGranted:          true,
        AccessReason:           "Valid pass found",
        GateID:                 req.GateID,
        Direction:              req.Direction,
    }

    if err := s.logRepo.Create(ctx, accessLog); err != nil {
        s.logger.Error().Err(err).Msg("Failed to create access log")
    }

    // 7. Вернуть результат с информацией о пользователе
    return &CheckAccessResponse{
        Success:       true,
        AccessGranted: true,
        User:          user,              // Возвращаем пользователя
        Vehicle:       vehicle,
        LicensePlate:  recognized.LicensePlate,
        Confidence:    recognized.Confidence,
        Reason:        "Access granted: valid pass",
        AccessLogID:   accessLog.ID,
    }, nil
}

// Вспомогательный метод для отказа в доступе
func (s *AccessService) denyAccess(ctx context.Context, plate, reason string, req *CheckAccessRequest) (*CheckAccessResponse, error) {
    // Записываем лог отказа (без user_id, если не нашли пользователя)
    accessLog := &domain.AccessLog{
        LicensePlate:  plate,
        AccessGranted: false,
        AccessReason:  reason,
        GateID:        req.GateID,
        Direction:     req.Direction,
    }
    s.logRepo.Create(ctx, accessLog)

    return &CheckAccessResponse{
        Success:       true,
        AccessGranted: false,
        LicensePlate:  plate,
        Reason:        reason,
    }, nil
}
```

## Domain Models (Go)

### User - центральная сущность

```go
// internal/domain/user.go

type User struct {
    ID           string
    Email        string
    PasswordHash string
    FullName     string
    Phone        string
    Role         UserRole
    IsActive     bool
    CreatedAt    time.Time
    UpdatedAt    time.Time
    LastLoginAt  *time.Time
}

type UserRole string

const (
    RoleAdmin UserRole = "admin"
    RoleUser  UserRole = "user"
    RoleGuard UserRole = "guard"
)

func (u *User) HasRole(role UserRole) bool {
    return u.Role == role
}

func (u *User) IsAdmin() bool {
    return u.Role == RoleAdmin
}

func (u *User) CanCreatePass() bool {
    return u.Role == RoleAdmin
}
```

### Vehicle - принадлежит пользователю

```go
// internal/domain/vehicle.go

type Vehicle struct {
    ID           string
    OwnerID      string    // ОБЯЗАТЕЛЬНОЕ поле - владелец
    LicensePlate string
    VehicleType  VehicleType
    Model        string
    Color        string
    IsActive     bool
    CreatedAt    time.Time
    UpdatedAt    time.Time

    // Связь с пользователем (загружается при необходимости)
    Owner        *User `json:"owner,omitempty"`
}

type VehicleType string

const (
    VehicleTypeCar        VehicleType = "car"
    VehicleTypeTruck      VehicleType = "truck"
    VehicleTypeMotorcycle VehicleType = "motorcycle"
    VehicleTypeBus        VehicleType = "bus"
    VehicleTypeOther      VehicleType = "other"
)

func (v *Vehicle) BelongsTo(userID string) bool {
    return v.OwnerID == userID
}
```

### Pass - выдается пользователю

```go
// internal/domain/pass.go

type Pass struct {
    ID          string
    UserID      string     // ПОЛЬЗОВАТЕЛЬ, которому выдан пропуск
    PassType    PassType
    ValidFrom   time.Time
    ValidUntil  *time.Time
    IsActive    bool
    RevokedAt   *time.Time
    RevokedBy   *string
    RevokeReason string
    CreatedAt   time.Time
    CreatedBy   *string
    UpdatedAt   time.Time

    // Связь с пользователем
    User        *User `json:"user,omitempty"`
}

type PassType string

const (
    PassTypePermanent PassType = "permanent"
    PassTypeTemporary PassType = "temporary"
)

func (p *Pass) IsValid(at time.Time) bool {
    if !p.IsActive {
        return false
    }

    if at.Before(p.ValidFrom) {
        return false
    }

    if p.PassType == PassTypeTemporary && p.ValidUntil != nil {
        if at.After(*p.ValidUntil) {
            return false
        }
    }

    return true
}

func (p *Pass) BelongsTo(userID string) bool {
    return p.UserID == userID
}
```

### AccessLog - фиксирует пользователя и транспорт

```go
// internal/domain/access_log.go

type AccessLog struct {
    ID                    string
    UserID                *string    // КТО получил доступ (главное!)
    VehicleID             *string    // ЧЕРЕЗ ЧТО (способ аутентификации)
    LicensePlate          string
    ImageURL              string
    RecognitionConfidence float64
    AccessGranted         bool
    AccessReason          string
    GateID                string
    Direction             Direction
    Timestamp             time.Time

    // Связи (загружаются при необходимости)
    User                  *User    `json:"user,omitempty"`
    Vehicle               *Vehicle `json:"vehicle,omitempty"`
}

type Direction string

const (
    DirectionIn  Direction = "IN"
    DirectionOut Direction = "OUT"
)

func (a *AccessLog) IsSuccessful() bool {
    return a.AccessGranted
}
```

## API Endpoints - обновленная структура

### Итерация 1: MVP

```
# Проверка доступа (для шлагбаума/охраны)
POST   /api/v1/access/check
Request:
{
  "image_base64": "...",
  "gate_id": "gate_001",
  "direction": "IN"
}

Response:
{
  "success": true,
  "access_granted": true,
  "user": {                          # Информация о пользователе
    "id": "uuid",
    "full_name": "Иван Иванов",
    "email": "ivan@example.com",
    "role": "user"
  },
  "vehicle": {
    "id": "uuid",
    "license_plate": "А123ВС777",
    "model": "Toyota Camry"
  },
  "license_plate": "А123ВС777",
  "confidence": 95.5,
  "reason": "Access granted: valid pass",
  "access_log_id": "uuid"
}

# Упрощенная регистрация пользователя (для MVP, без JWT)
POST   /api/v1/users/register
{
  "email": "user@example.com",
  "password": "password123",
  "full_name": "Иван Иванов",
  "phone": "+7 999 123 45 67"
}

# Добавление автомобиля пользователю
POST   /api/v1/users/{user_id}/vehicles
{
  "license_plate": "А123ВС777",
  "model": "Toyota Camry",
  "color": "black",
  "vehicle_type": "car"
}

# Создание пропуска для пользователя (админ)
POST   /api/v1/passes
{
  "user_id": "uuid",
  "pass_type": "permanent"
}

# История проездов (для анализа)
GET    /api/v1/access/logs?user_id={id}    # По пользователю
GET    /api/v1/access/logs?vehicle_id={id} # По автомобилю
GET    /api/v1/access/logs                 # Все логи (admin)
```

### Итерация 2: JWT аутентификация и личный кабинет

```
# Аутентификация
POST   /api/v1/auth/login
POST   /api/v1/auth/logout
POST   /api/v1/auth/refresh

# Личный кабинет
GET    /api/v1/users/me                    # Мой профиль
PUT    /api/v1/users/me                    # Обновить профиль
GET    /api/v1/users/me/pass               # Мой пропуск
GET    /api/v1/users/me/vehicles           # Мои автомобили
POST   /api/v1/users/me/vehicles           # Добавить автомобиль
DELETE /api/v1/users/me/vehicles/{id}      # Удалить автомобиль
GET    /api/v1/users/me/access-logs        # Моя история проездов
```

## Обновленный план разработки

### Итерация 1: MVP (3-4 недели)

**Цель**: Полная функциональность проверки доступа с пользователями

#### Неделя 1: База данных и Domain

1. **Миграция БД** - создать ВСЕ таблицы (users, vehicles, passes, access_logs, refresh_tokens)
2. **Domain models** - User, Vehicle, Pass, AccessLog
3. **Domain errors** - typed errors для бизнес-логики

#### Неделя 2: Repository и Infrastructure

4. **Repository layer**:
   - UserRepository (CRUD пользователей)
   - VehicleRepository (с фильтрацией по owner_id)
   - PassRepository (GetActivePassByUser - ключевой метод!)
   - AccessLogRepository (с фильтрацией по user_id и vehicle_id)

5. **Infrastructure**:
   - Config loader
   - Logger (zerolog)
   - Password hashing (bcrypt)

#### Неделя 3: ML сервис

6. **Python ML сервис**:
   - FastAPI приложение
   - EasyOCR интеграция
   - Endpoint распознавания

7. **Go ML клиент**:
   - HTTP клиент с retry
   - Error handling
   - Timeout management

#### Неделя 4: Use Cases и API

8. **Use Cases**:
   - **AccessService.CheckAccess()** - ГЛАВНЫЙ метод с правильной логикой
   - UserService (регистрация пользователей)
   - VehicleService (добавление автомобилей)
   - PassService (создание пропусков)

9. **HTTP Handlers**:
   - access_handler.go
   - user_handler.go
   - vehicle_handler.go
   - pass_handler.go

10. **Main application** - собираем все вместе

### Итерация 2: JWT и Frontend (2 недели)

1. JWT authentication
2. Auth middleware
3. Vue 3 + Pinia frontend
4. Личный кабинет

### Итерация 3: Временные пропуска (1 неделя)

1. Временные ограничения
2. RBAC
3. Cron для истечения
4. UI управления

### Итерация 4: История (1 неделя)

1. Pass history таблица
2. Analytics endpoints
3. Отчеты и графики

## Ключевые преимущества новой архитектуры

### 1. Правильная бизнес-логика
- ✅ Пользователь - центральная сущность
- ✅ Автомобиль - способ аутентификации
- ✅ Пропуск привязан к пользователю, а не к автомобилю

### 2. Гибкость
- ✅ Один пользователь может иметь несколько автомобилей
- ✅ Смена автомобиля не требует перевыпуска пропуска
- ✅ История привязана к пользователю

### 3. Безопасность
- ✅ Отозвать пропуск пользователя = закрыть доступ для всех его автомобилей
- ✅ Удаление пользователя удаляет все связанные данные (CASCADE)
- ✅ Аудит: всегда известно, кто получил доступ

### 4. Аналитика
- ✅ Статистика по пользователям
- ✅ Анализ частоты посещений
- ✅ Отчеты по пользователям, а не по номерам

## Следующий шаг

Обновить миграцию БД с новой схемой, где users - первая таблица, а vehicles обязательно связаны с пользователем.
