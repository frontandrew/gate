-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create enum types
CREATE TYPE vehicle_type_enum AS ENUM ('car', 'truck', 'motorcycle', 'bus', 'other');
CREATE TYPE direction_enum AS ENUM ('IN', 'OUT');
CREATE TYPE user_role_enum AS ENUM ('admin', 'user', 'guard');
CREATE TYPE pass_type_enum AS ENUM ('permanent', 'temporary');

-- ============================================================================
-- USERS TABLE - Центральная сущность системы
-- ============================================================================
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    phone VARCHAR(20),
    role user_role_enum NOT NULL DEFAULT 'user',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMP,

    CONSTRAINT email_format CHECK (email ~ '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$')
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_is_active ON users(is_active) WHERE is_active = true;

COMMENT ON TABLE users IS 'Пользователи системы - центральная сущность';
COMMENT ON COLUMN users.email IS 'Email пользователя (уникальный идентификатор)';
COMMENT ON COLUMN users.role IS 'Роль: admin - администратор, user - обычный пользователь, guard - охранник';

-- ============================================================================
-- VEHICLES TABLE - Автомобили пользователей (способ аутентификации)
-- ============================================================================
CREATE TABLE IF NOT EXISTS vehicles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
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
CREATE INDEX idx_vehicles_created_at ON vehicles(created_at DESC);

COMMENT ON TABLE vehicles IS 'Автомобили пользователей - способ аутентификации в системе';
COMMENT ON COLUMN vehicles.owner_id IS 'Владелец автомобиля (ОБЯЗАТЕЛЬНАЯ связь с users)';
COMMENT ON COLUMN vehicles.license_plate IS 'Номер автомобиля (уникальный)';

-- ============================================================================
-- PASSES TABLE - Пропуска выдаются пользователям
-- ============================================================================
CREATE TABLE IF NOT EXISTS passes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    pass_type pass_type_enum NOT NULL DEFAULT 'permanent',
    valid_from TIMESTAMP NOT NULL DEFAULT NOW(),
    valid_until TIMESTAMP,
    is_active BOOLEAN NOT NULL DEFAULT true,
    revoked_at TIMESTAMP,
    revoked_by UUID REFERENCES users(id),
    revoke_reason VARCHAR(500),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_dates_check CHECK (valid_until IS NULL OR valid_until > valid_from)
);

CREATE INDEX idx_passes_user_id ON passes(user_id);
CREATE INDEX idx_passes_valid_dates ON passes(valid_from, valid_until);
CREATE INDEX idx_passes_is_active ON passes(is_active) WHERE is_active = true;
CREATE INDEX idx_passes_pass_type ON passes(pass_type);

-- Создаем частичный уникальный индекс для активных пропусков
-- Один пользователь может иметь только один активный пропуск
CREATE UNIQUE INDEX idx_passes_unique_active
    ON passes(user_id)
    WHERE is_active = true;

COMMENT ON TABLE passes IS 'Пропуска выдаются пользователям (не автомобилям!)';
COMMENT ON COLUMN passes.user_id IS 'Пользователь, которому выдан пропуск';
COMMENT ON COLUMN passes.pass_type IS 'Тип пропуска: permanent - постоянный, temporary - временный';

-- ============================================================================
-- ACCESS_LOGS TABLE - История доступа (КТО через ЧТО)
-- ============================================================================
CREATE TABLE IF NOT EXISTS access_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id),
    vehicle_id UUID REFERENCES vehicles(id),
    license_plate VARCHAR(20) NOT NULL,
    image_url VARCHAR(500),
    recognition_confidence DECIMAL(5,2) CHECK (recognition_confidence >= 0 AND recognition_confidence <= 100),
    access_granted BOOLEAN NOT NULL,
    access_reason VARCHAR(255),
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
CREATE INDEX idx_access_logs_user_timestamp ON access_logs(user_id, timestamp DESC);

COMMENT ON TABLE access_logs IS 'История доступа: КТО (user) получил доступ ЧЕРЕЗ ЧТО (vehicle)';
COMMENT ON COLUMN access_logs.user_id IS 'Главная информация - какой пользователь получил доступ';
COMMENT ON COLUMN access_logs.vehicle_id IS 'Вспомогательная информация - через какой автомобиль';
COMMENT ON COLUMN access_logs.recognition_confidence IS 'Уверенность распознавания номера (0-100%)';

-- ============================================================================
-- REFRESH_TOKENS TABLE - Для JWT аутентификации
-- ============================================================================
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMP
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);

COMMENT ON TABLE refresh_tokens IS 'Refresh токены для JWT аутентификации';

-- ============================================================================
-- GATES TABLE - Ворота/шлагбаумы (опционально)
-- ============================================================================
CREATE TABLE IF NOT EXISTS gates (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    location VARCHAR(255),
    gate_type VARCHAR(20) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT gate_type_check CHECK (gate_type IN ('entry', 'exit', 'both'))
);

COMMENT ON TABLE gates IS 'Ворота и шлагбаумы на охраняемой территории';

-- ============================================================================
-- FUNCTIONS AND TRIGGERS
-- ============================================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for updated_at
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_vehicles_updated_at
    BEFORE UPDATE ON vehicles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_passes_updated_at
    BEFORE UPDATE ON passes
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
