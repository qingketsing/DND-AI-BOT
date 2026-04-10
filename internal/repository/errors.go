package repository

import "errors"

var (
	ErrSessionNotFound     = errors.New("session not found")
	ErrGameStateNotFound   = errors.New("game state not found")
	ErrEncounterNotFound   = errors.New("encounter not found")
	ErrUserNotFound        = errors.New("user not found")
	ErrAuthSessionNotFound = errors.New("auth session not found")

	ErrCacheMiss           = errors.New("cache miss")
	ErrCacheNotFoundMarker = errors.New("cache not found marker")
)
