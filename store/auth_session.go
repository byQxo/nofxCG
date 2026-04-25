package store

import (
	"time"

	"gorm.io/gorm"
)

// AuthSession 持久化会话表。
// 只保存 refresh token 哈希，不保存原始令牌，降低数据库泄露后的可重放风险。
type AuthSession struct {
	SessionID        string     `gorm:"column:session_id;primaryKey" json:"session_id"`
	UserID           string     `gorm:"column:user_id;not null;index" json:"user_id"`
	RefreshTokenHash string     `gorm:"column:refresh_token_hash;not null" json:"-"`
	AccessExpiresAt  time.Time  `gorm:"column:access_expires_at;not null;index" json:"access_expires_at"`
	RefreshExpiresAt time.Time  `gorm:"column:refresh_expires_at;not null;index" json:"refresh_expires_at"`
	RevokedAt        *time.Time `gorm:"column:revoked_at;index" json:"revoked_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (AuthSession) TableName() string { return "auth_sessions" }

type AuthSessionStore struct {
	db *gorm.DB
}

func NewAuthSessionStore(db *gorm.DB) *AuthSessionStore {
	return &AuthSessionStore{db: db}
}

func (s *AuthSessionStore) initTables() error {
	return s.db.AutoMigrate(&AuthSession{})
}

func (s *AuthSessionStore) Upsert(session *AuthSession) error {
	return s.db.Save(session).Error
}

func (s *AuthSessionStore) Get(sessionID string) (*AuthSession, error) {
	var session AuthSession
	if err := s.db.Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *AuthSessionStore) Revoke(sessionID string, revokedAt time.Time) error {
	return s.db.Model(&AuthSession{}).
		Where("session_id = ?", sessionID).
		Updates(map[string]any{
			"revoked_at":  revokedAt.UTC(),
			"updated_at":  time.Now().UTC(),
		}).Error
}

func (s *AuthSessionStore) DeleteExpired(now time.Time) error {
	return s.db.Where("refresh_expires_at < ? OR (revoked_at IS NOT NULL AND revoked_at < ?)", now.UTC(), now.UTC().Add(-24*time.Hour)).
		Delete(&AuthSession{}).Error
}
