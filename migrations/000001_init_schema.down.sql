-- Drop triggers (в обратном порядке создания)
DROP TRIGGER IF EXISTS update_passes_updated_at ON passes;
DROP TRIGGER IF EXISTS update_vehicles_updated_at ON vehicles;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables (в обратном порядке создания, учитывая зависимости)
DROP TABLE IF EXISTS gates;
DROP TABLE IF EXISTS blacklist;
DROP TABLE IF EXISTS whitelist;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS access_logs;
DROP TABLE IF EXISTS pass_vehicles;
DROP TABLE IF EXISTS passes;
DROP TABLE IF EXISTS vehicles;
DROP TABLE IF EXISTS users;

-- Drop enum types
DROP TYPE IF EXISTS pass_type_enum;
DROP TYPE IF EXISTS user_role_enum;
DROP TYPE IF EXISTS direction_enum;
DROP TYPE IF EXISTS vehicle_type_enum;

-- Drop UUID extension (commented out as it might be used by other tables)
-- DROP EXTENSION IF EXISTS "uuid-ossp";
