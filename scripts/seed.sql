-- ============================================================================
-- SEED DATA для тестирования системы GATE
-- ============================================================================

-- Очистка существующих данных (осторожно! удаляет все данные)
TRUNCATE TABLE access_logs, refresh_tokens, passes, vehicles, users, gates RESTART IDENTITY CASCADE;

-- ============================================================================
-- 1. USERS - Тестовые пользователи
-- ============================================================================

-- Пароль для всех: "password123" (bcrypt hash с cost=12)
-- Можно сгенерировать: echo -n "password123" | bcrypt-cli -c 12

INSERT INTO users (id, email, password_hash, full_name, phone, role, is_active) VALUES
-- Администратор
('00000000-0000-0000-0000-000000000001', 'admin@gate.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIpPqHqKmu', 'Администратор Системы', '+7 999 000 00 01', 'admin', true),

-- Обычные пользователи
('00000000-0000-0000-0000-000000000002', 'ivan@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIpPqHqKmu', 'Иван Иванов', '+7 999 111 11 11', 'user', true),
('00000000-0000-0000-0000-000000000003', 'maria@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIpPqHqKmu', 'Мария Петрова', '+7 999 222 22 22', 'user', true),
('00000000-0000-0000-0000-000000000004', 'alex@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIpPqHqKmu', 'Александр Сидоров', '+7 999 333 33 33', 'user', true),

-- Охранник
('00000000-0000-0000-0000-000000000005', 'guard@gate.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIpPqHqKmu', 'Сергей Охранников', '+7 999 444 44 44', 'guard', true),

-- Неактивный пользователь
('00000000-0000-0000-0000-000000000006', 'blocked@example.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIpPqHqKmu', 'Заблокированный Пользователь', '+7 999 555 55 55', 'user', false);

-- ============================================================================
-- 2. VEHICLES - Автомобили пользователей
-- ============================================================================

INSERT INTO vehicles (id, owner_id, license_plate, vehicle_type, model, color, is_active) VALUES
-- Автомобили Ивана Иванова
('10000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000002', 'А123ВС777', 'car', 'Toyota Camry', 'black', true),
('10000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000002', 'В456ЕК777', 'car', 'BMW X5', 'white', true),

-- Автомобиль Марии Петровой
('10000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000003', 'С789МН777', 'car', 'Mercedes-Benz E-Class', 'silver', true),

-- Автомобиль Александра Сидорова
('10000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000004', 'Т001ОР777', 'truck', 'Ford F-150', 'blue', true),

-- Автомобиль администратора
('10000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'Е999УУ777', 'car', 'Tesla Model S', 'red', true),

-- Неактивный автомобиль
('10000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000002', 'Н111АА777', 'car', 'Lada Vesta', 'gray', false);

-- ============================================================================
-- 3. PASSES - Пропуска для пользователей
-- ============================================================================

INSERT INTO passes (id, user_id, pass_type, valid_from, valid_until, is_active, created_by) VALUES
-- Постоянный пропуск для Ивана Иванова
('20000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000002', 'permanent', NOW(), NULL, true, '00000000-0000-0000-0000-000000000001'),

-- Постоянный пропуск для Марии Петровой
('20000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000003', 'permanent', NOW(), NULL, true, '00000000-0000-0000-0000-000000000001'),

-- Временный пропуск для Александра Сидорова (действителен 30 дней)
('20000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000004', 'temporary', NOW(), NOW() + INTERVAL '30 days', true, '00000000-0000-0000-0000-000000000001'),

-- Постоянный пропуск для администратора
('20000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'permanent', NOW(), NULL, true, '00000000-0000-0000-0000-000000000001'),

-- Постоянный пропуск для охранника
('20000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000005', 'permanent', NOW(), NULL, true, '00000000-0000-0000-0000-000000000001');

-- ============================================================================
-- 4. GATES - Ворота на территории
-- ============================================================================

INSERT INTO gates (id, name, location, gate_type, is_active) VALUES
('gate_001', 'Главный въезд', 'Северная сторона', 'both', true),
('gate_002', 'Служебный въезд', 'Западная сторона', 'entry', true),
('gate_003', 'Грузовой въезд', 'Восточная сторона', 'entry', true),
('gate_004', 'Аварийный выезд', 'Южная сторона', 'exit', true);

-- ============================================================================
-- 5. ACCESS_LOGS - Тестовые записи о проездах
-- ============================================================================

INSERT INTO access_logs (
    user_id,
    vehicle_id,
    license_plate,
    recognition_confidence,
    access_granted,
    access_reason,
    gate_id,
    direction,
    timestamp
) VALUES
-- Успешные проезды
('00000000-0000-0000-0000-000000000002', '10000000-0000-0000-0000-000000000001', 'А123ВС777', 98.5, true, 'Valid pass found', 'gate_001', 'IN', NOW() - INTERVAL '2 hours'),
('00000000-0000-0000-0000-000000000003', '10000000-0000-0000-0000-000000000003', 'С789МН777', 95.2, true, 'Valid pass found', 'gate_001', 'IN', NOW() - INTERVAL '1 hour'),
('00000000-0000-0000-0000-000000000004', '10000000-0000-0000-0000-000000000004', 'Т001ОР777', 97.8, true, 'Valid pass found', 'gate_003', 'IN', NOW() - INTERVAL '30 minutes'),
('00000000-0000-0000-0000-000000000002', '10000000-0000-0000-0000-000000000002', 'В456ЕК777', 99.1, true, 'Valid pass found', 'gate_001', 'IN', NOW() - INTERVAL '15 minutes'),

-- Отказы в доступе
(NULL, NULL, 'Х999ХХ999', 89.5, false, 'Vehicle not found', 'gate_001', 'IN', NOW() - INTERVAL '3 hours'),
(NULL, NULL, 'Ж888ЖЖ888', 92.3, false, 'Vehicle not found', 'gate_002', 'IN', NOW() - INTERVAL '1 hour 30 minutes');

-- ============================================================================
-- Вывод статистики
-- ============================================================================

SELECT
    '=== ИТОГИ ЗАПОЛНЕНИЯ БД ===' as info,
    (SELECT COUNT(*) FROM users) as users_count,
    (SELECT COUNT(*) FROM vehicles) as vehicles_count,
    (SELECT COUNT(*) FROM passes) as passes_count,
    (SELECT COUNT(*) FROM gates) as gates_count,
    (SELECT COUNT(*) FROM access_logs) as access_logs_count;

-- Пользователи с пропусками
SELECT
    '=== ПОЛЬЗОВАТЕЛИ С ПРОПУСКАМИ ===' as info;

SELECT
    u.full_name,
    u.email,
    u.role,
    COUNT(v.id) as vehicles_count,
    p.pass_type,
    p.valid_from,
    p.valid_until,
    p.is_active as pass_active
FROM users u
LEFT JOIN vehicles v ON v.owner_id = u.id
LEFT JOIN passes p ON p.user_id = u.id
WHERE u.is_active = true
GROUP BY u.id, u.full_name, u.email, u.role, p.pass_type, p.valid_from, p.valid_until, p.is_active
ORDER BY u.full_name;

-- Статистика проездов
SELECT
    '=== СТАТИСТИКА ПРОЕЗДОВ ===' as info;

SELECT
    access_granted,
    COUNT(*) as count,
    ROUND(AVG(recognition_confidence), 2) as avg_confidence
FROM access_logs
GROUP BY access_granted;
