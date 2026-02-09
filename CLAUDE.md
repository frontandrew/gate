# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GATE is a gate access control system using automatic vehicle license plate recognition. The system uses a **user-centric architecture** where users are the central entity, and vehicles are authentication methods.

**Core Technology Stack:**
- Backend: Go 1.22+ (Clean Architecture)
- Database: PostgreSQL 16
- Cache: Redis 7 (infrastructure ready, not yet implemented)
- ML Service: Python FastAPI + EasyOCR for plate recognition
- Infrastructure: Docker Compose

## Common Commands

### Development Workflow
```bash
# Start infrastructure (PostgreSQL, Redis)
make docker-up

# Apply database migrations
make migrate-up

# Run API server (in Docker)
make run

# Run API in background
make run-d

# Stop all services
make docker-down
```

### Testing & Quality
```bash
# Run all tests
make test

# Run tests for specific package
docker run --rm -v $(pwd):/app -w /app golang:1.22-alpine \
  go test -v ./internal/delivery/http -run TestAuthHandler

# Run linter (golangci-lint)
make lint

# Type checking
make typecheck

# Test coverage report
docker run --rm -v $(pwd):/app -w /app golang:1.22-alpine \
  go test -coverprofile=coverage.out ./internal/delivery/http
```

### Database Operations
```bash
# Create new migration
make migrate-create name=add_new_table

# Rollback migrations
make migrate-down

# Seed test data
make seed
```

## Architecture Overview

### Clean Architecture Layers

The project follows Clean Architecture with strict dependency rules:

```
cmd/api/          → Application entry point
internal/
├── domain/       → Business entities and rules (no external dependencies)
├── usecase/      → Business logic (depends on domain & repository interfaces)
├── repository/   → Data access interfaces + PostgreSQL implementations
├── delivery/     → HTTP handlers (depends on usecase interfaces)
├── infrastructure/ → External services (ML, storage, notifications)
└── pkg/          → Shared utilities (config, logger, jwt, database)
```

**Dependency Flow:** `delivery → usecase → repository → domain`

### Core Business Logic: Access Control

The access check follows a **prioritized 3-tier security model**:

1. **Whitelist (Highest Priority)** - Immediate access granted (emergency vehicles, VIP)
2. **Blacklist (Second Priority)** - Immediate access denied (stolen vehicles, banned)
3. **Standard Check (Third Priority)** - User → Vehicle → Active Pass validation

**Access Flow:**
```
License Plate Recognition (ML)
    ↓
Whitelist Check → GRANT (if found)
    ↓
Blacklist Check → DENY (if found)
    ↓
Find Vehicle by plate
    ↓
Find Owner (User) via vehicle.owner_id
    ↓
Find Active Passes for User + Vehicle
    ↓
Validate Pass (time constraints, active status)
    ↓
GRANT or DENY based on valid pass
```

### User-Centric Design Principles

**Critical:** The user is the central entity. Vehicles are authentication methods, NOT independent entities.

- ✅ Vehicles MUST have an owner (owner_id NOT NULL)
- ✅ Passes are issued to users, not vehicles
- ✅ Access logs record which user accessed (user_id) via which vehicle (vehicle_id)
- ✅ Statistics and analytics are user-based

**Key Repository Method:**
```go
// Most critical method for access control
GetActivePassesByUserAndVehicle(ctx, userID, vehicleID) ([]*Pass, error)
```

### Service Layer Pattern

All use cases follow dependency injection with interfaces:

```go
// Service constructor pattern
func NewService(
    repo repository.SomeRepository,
    logger logger.Logger,
    // ... other dependencies via interfaces
) *Service {
    return &Service{
        repo: repo,
        logger: logger,
    }
}
```

**Available Services:**
- `access.Service` - Core access control logic (CheckAccess, GetLogs)
- `auth.Service` - User authentication (Register, Login, JWT)
- `vehicle.Service` - Vehicle management
- `pass.Service` - Pass management (Create, Revoke)

## Testing Patterns

### Table-Driven Tests with Mocks

All HTTP handler tests use table-driven pattern with testify/mock:

```go
tests := []struct {
    name           string
    requestBody    interface{}
    mockSetup      func(*MockService)
    expectedStatus int
    checkResponse  func(*testing.T, map[string]interface{})
}{
    {
        name: "success case",
        requestBody: Request{...},
        mockSetup: func(m *MockService) {
            m.On("Method", mock.Anything, mock.Anything).Return(result, nil)
        },
        expectedStatus: http.StatusOK,
        checkResponse: func(t *testing.T, resp map[string]interface{}) {
            assert.True(t, resp["success"].(bool))
        },
    },
}
```

**Test Helpers:** Use functions from `internal/delivery/http/test_helpers.go`:
- `CreateTestUser()` - Generate test user
- `CreateAuthContext()` - Context with user_id
- `CreateTestJWTToken()` - Valid JWT for testing

**Mock Services:** Located in each `*_test.go` file:
- `MockAuthService`
- `MockVehicleService`
- `MockPassService`
- `MockAccessService`

## Database Schema Essentials

### Critical Tables and Relationships

```sql
users (central entity)
  ↓ (owner_id)
vehicles (MUST have owner, unique license_plate)
  ↓ (pass_vehicles many-to-many)
passes (belongs to user, has time constraints)

-- Separate priority tables
whitelist (license_plate, reason, expires_at)
blacklist (license_plate, reason, expires_at)

-- Audit trail
access_logs (user_id, vehicle_id, access_granted, reason)
refresh_tokens (for JWT auth)
```

**Important Constraints:**
- `vehicles.owner_id` is NOT NULL (enforces user-centric model)
- `vehicles.license_plate` is UNIQUE
- `passes` have `valid_from` and `valid_until` (for temporary passes)
- All IDs are UUIDs

### Migration Pattern

Migrations use golang-migrate with up/down files:
```
migrations/
  000001_init_schema.up.sql    # Create tables
  000001_init_schema.down.sql  # Drop tables
```

## ML Service Integration

The Python ML service (`gate-ml/`) runs independently:

```bash
# ML service is in gate-ml/ directory
# Runs on port 8001 by default
# Provides POST /recognize endpoint
```

**Go ML Client:** `internal/infrastructure/ml/client.go`
- HTTP-based communication
- Handles timeouts and retries
- Returns license plate + confidence score

**Minimum Confidence:** Configured via `ML_MIN_CONFIDENCE` env var (default: 0.7)

## Configuration

All configuration via environment variables (`.env` file):

**Database:**
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`

**JWT:**
- `JWT_SECRET` - MUST change in production
- `JWT_ACCESS_EXPIRY` - Access token lifetime (seconds)
- `JWT_REFRESH_EXPIRY` - Refresh token lifetime (seconds)

**ML Service:**
- `ML_SERVICE_URL` - Python ML service URL
- `ML_TIMEOUT` - Request timeout
- `ML_MIN_CONFIDENCE` - Minimum recognition confidence

**Server:**
- `SERVER_PORT` - API server port (default: 8080)
- `SERVER_HOST` - Bind address

## API Endpoints Structure

**Auth:**
- `POST /api/v1/auth/register` - User registration
- `POST /api/v1/auth/login` - Login (returns JWT)
- `POST /api/v1/auth/logout` - Logout
- `POST /api/v1/auth/refresh` - Refresh JWT token

**Vehicles:**
- `POST /api/v1/vehicles` - Create vehicle
- `GET /api/v1/vehicles/me` - Get my vehicles
- `GET /api/v1/vehicles/:id` - Get vehicle by ID

**Passes:**
- `POST /api/v1/passes` - Create pass (admin/guard)
- `GET /api/v1/passes/me` - Get my passes
- `GET /api/v1/passes/:id` - Get pass by ID
- `DELETE /api/v1/passes/:id/revoke` - Revoke pass (admin/guard)

**Access Control:**
- `POST /api/v1/access/check` - Check access (main endpoint)
- `GET /api/v1/access/logs` - Get all access logs (admin/guard)
- `GET /api/v1/access/logs/vehicle/:id` - Logs for vehicle
- `GET /api/v1/access/me/logs` - My access logs

## Code Style & Patterns

### Error Handling

Use typed domain errors from `internal/domain/errors.go`:
```go
domain.ErrNotFound
domain.ErrAlreadyExists
domain.ErrUnauthorized
domain.ErrForbidden
domain.ErrInvalidInput
```

### Logging

Logger is injected via `logger.Logger` interface:
```go
logger.Info("message", map[string]interface{}{
    "key": "value",
})
logger.Error("error message", map[string]interface{}{
    "error": err.Error(),
})
```

### UUID Handling

Always use `github.com/google/uuid` package:
```go
id := uuid.New()                    // Generate new UUID
uid, err := uuid.Parse(idString)    // Parse string to UUID
```

## Development Iteration Status

**Current Status:** Iteration 1 (MVP) - Core functionality complete

**Completed:**
- ✅ Database schema with all tables
- ✅ Domain models (User, Vehicle, Pass, AccessLog, Whitelist, Blacklist)
- ✅ Repository layer (PostgreSQL implementations)
- ✅ Use case layer (auth, vehicle, pass, access)
- ✅ HTTP handlers with comprehensive tests
- ✅ JWT authentication
- ✅ Access control with whitelist/blacklist
- ✅ ML service integration (Python + Go client)

**Planned:**
- Frontend (Vue 3 + Pinia)
- Enhanced analytics and reporting
- WebSocket notifications for real-time gate control

## Important Notes for Development

1. **Always preserve user-centric logic**: Never create vehicles without owners, passes without users.

2. **Access control priority is critical**: Whitelist → Blacklist → Standard check. Never change this order.

3. **Test coverage is important**: Maintain 75%+ coverage. Use table-driven tests with mocks.

4. **Use interfaces for dependencies**: All services receive dependencies via interfaces, enabling testability.

5. **Database operations**: All repository methods accept `context.Context` as first parameter.

6. **JWT middleware**: Protected endpoints use `middleware.Auth()` to extract user_id from JWT.

7. **HTTP response format**: All responses follow structure:
   ```json
   {
     "success": true/false,
     "data": {...} or "error": "message"
   }
   ```
