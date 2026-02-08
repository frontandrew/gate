package domain

import "errors"

// Доменные ошибки - используются во всех слоях приложения

// User errors
var (
	ErrUserNotFound     = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidEmail     = errors.New("invalid email")
	ErrInvalidPassword  = errors.New("invalid password")
	ErrInvalidUserData  = errors.New("invalid user data")
	ErrInvalidRole      = errors.New("invalid user role")
	ErrUserInactive     = errors.New("user is inactive")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// Vehicle errors
var (
	ErrVehicleNotFound      = errors.New("vehicle not found")
	ErrVehicleAlreadyExists = errors.New("vehicle already exists")
	ErrInvalidLicensePlate  = errors.New("invalid license plate")
	ErrInvalidVehicleData   = errors.New("invalid vehicle data")
)

// Pass errors
var (
	ErrPassNotFound       = errors.New("pass not found")
	ErrInvalidPassData    = errors.New("invalid pass data")
	ErrInvalidPassType    = errors.New("invalid pass type")
	ErrInvalidDateRange   = errors.New("invalid date range")
	ErrPassExpired        = errors.New("pass expired")
	ErrPassNotActive      = errors.New("pass is not active")
	ErrPassAlreadyRevoked = errors.New("pass already revoked")
	ErrNoValidPass        = errors.New("no valid pass found")
)

// PassVehicle errors
var (
	ErrPassVehicleNotFound      = errors.New("pass-vehicle relation not found")
	ErrPassVehicleAlreadyExists = errors.New("pass-vehicle relation already exists")
	ErrInvalidPassVehicleData   = errors.New("invalid pass-vehicle data")
)

// AccessLog errors
var (
	ErrAccessLogNotFound     = errors.New("access log not found")
	ErrInvalidAccessLogData  = errors.New("invalid access log data")
	ErrInvalidDirection      = errors.New("invalid direction")
	ErrInvalidConfidence     = errors.New("invalid recognition confidence")
)

// Authorization errors
var (
	ErrUnauthorized     = errors.New("unauthorized")
	ErrForbidden        = errors.New("forbidden")
	ErrTokenExpired     = errors.New("token expired")
	ErrInvalidToken     = errors.New("invalid token")
)

// General errors
var (
	ErrInternal      = errors.New("internal server error")
	ErrNotFound      = errors.New("not found")
	ErrBadRequest    = errors.New("bad request")
	ErrConflict      = errors.New("conflict")
)
