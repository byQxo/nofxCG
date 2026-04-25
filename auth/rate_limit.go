package auth

import (
	"time"

	"nofx/bootstrap"
	"nofx/store"
)

const loginRateLimitConfigKey = "security.login_rate_limit"

type loginRateLimitState struct {
	WindowStartedAt time.Time `json:"window_started_at"`
	FailCount       int       `json:"fail_count"`
	LockedUntil     time.Time `json:"locked_until"`
}

func canAttemptAdminLogin(st *store.Store) (bool, error) {
	state, err := loadRateLimitState(st)
	if err != nil {
		return false, err
	}
	if time.Now().UTC().Before(state.LockedUntil) {
		return false, nil
	}
	return true, nil
}

func recordAdminLoginFailure(st *store.Store) (time.Time, error) {
	now := time.Now().UTC()
	state, err := loadRateLimitState(st)
	if err != nil {
		return time.Time{}, err
	}
	if state.WindowStartedAt.IsZero() || now.Sub(state.WindowStartedAt) > time.Minute {
		state.WindowStartedAt = now
		state.FailCount = 0
	}
	state.FailCount++
	if state.FailCount >= 5 {
		state.LockedUntil = now.Add(10 * time.Minute)
	}
	if err := bootstrap.WriteRateLimitState(st, loginRateLimitConfigKey, state); err != nil {
		return time.Time{}, err
	}
	return state.LockedUntil, nil
}

func clearAdminLoginFailures(st *store.Store) error {
	return bootstrap.WriteRateLimitState(st, loginRateLimitConfigKey, loginRateLimitState{})
}

func loadRateLimitState(st *store.Store) (*loginRateLimitState, error) {
	state := &loginRateLimitState{}
	if err := bootstrap.ReadRateLimitState(st, loginRateLimitConfigKey, state); err != nil {
		return nil, err
	}
	return state, nil
}
