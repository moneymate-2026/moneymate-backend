	package redis

	import (
		"context"
		"fmt"
		"time"

		"github.com/moneymate-2026/moneymate-backend/auth/internal/domain"
		"github.com/redis/go-redis/v9"
	)

	type redisStore struct {
		client *redis.Client
	}

	// returns a new store
	func NewStore(c *redis.Client) domain.Store {
		return &redisStore{
			client: c,
		}
	}

	// upgrade token version
	func (r *redisStore) UpgradeTokenVersion(ctx context.Context, userID string) error {
		key := fmt.Sprintf("auth:user:%s:version", userID)
		err := r.client.Incr(ctx, key).Err()
		if err != nil {
			return fmt.Errorf("redis increment version error: %w", err)
		}
		return nil
	}

	// get token version
	func (r *redisStore) GetTokenVersion(ctx context.Context, userID string) (int64, error) {
		key := fmt.Sprintf("auth:user:%s:version", userID)
		version, err := r.client.Get(ctx, key).Int64()
		if err == redis.Nil {
			return 0, nil
		}
		if err != nil {
			return 0, fmt.Errorf("getting token version: %w", err)
		}
		return version, nil
	}

	func (r *redisStore) ClaimRefreshToken(ctx context.Context, tokenID string, ttl time.Duration) (bool, error) {
		key := fmt.Sprintf("claim:%s", tokenID)

		set, err := r.client.SetNX(ctx, key, "claimed", ttl).Result()
		if err != nil {
			return false, fmt.Errorf("redis setnx error: %w", err)
		}
		return !set, nil
	}



// ── OTP ────────────────────────────────────────────────

func (r *redisStore)    SetRegistrationOTP (ctx context.Context, email, otpHash string, ttl time.Duration) error {
    key := otpKey(email)
    if err := r.client.Set(ctx, key, otpHash, ttl).Err(); err != nil {
        return fmt.Errorf("redis set registration otp: %w", err)
    }
    return nil
}

func (r *redisStore) GetRegistrationOTP(ctx context.Context, email string) (string, bool, error) {
    key := otpKey(email)
    hash, err := r.client.Get(ctx, key).Result()
    if err == redis.Nil {
        return "", false, nil
    }
    if err != nil {
        return "", false, fmt.Errorf("redis get registration otp: %w", err)
    }
    return hash, true, nil
}

func (r *redisStore) DeleteRegistrationOTP(ctx context.Context, email string) error {
    key := otpKey(email)
    if err := r.client.Del(ctx, key).Err(); err != nil {
        return fmt.Errorf("redis delete registration otp: %w", err)
    }
    return nil
}   

func (r *redisStore) IncrementOTPAttempts(ctx context.Context, email string, ttl time.Duration) (int64, error) {
    key := otpAttemptsKey(email)

    count, err := r.client.Incr(ctx, key).Result()
    if err != nil {
        return 0, fmt.Errorf("redis increment otp attempts: %w", err)
    }
    if count == 1 {
        if err := r.client.Expire(ctx, key, ttl).Err(); err != nil {
            return count, fmt.Errorf("redis set otp attempts ttl: %w", err)
        }
    }
    return count, nil
}

func (r *redisStore) TrySetResendCooldown(ctx context.Context, email string, ttl time.Duration) (bool, time.Duration, error) {
    key := otpCooldownKey(email)
    
    ok, err := r.client.SetNX(ctx, key, "1", ttl).Result()
    if err != nil {
        return false, 0, fmt.Errorf("redis set resend cooldown: %w", err)
    }
    if ok {
        return true, ttl, nil 
    }
    remainingTTL, err := r.client.TTL(ctx, key).Result()
    if err != nil {
        return false, 0, fmt.Errorf("redis get cooldown ttl: %w", err)
    }
    return false, remainingTTL, nil
}


func (r *redisStore) MarkEmailVerified(ctx context.Context, email string, ttl time.Duration) error {
    key := emailVerifiedKey(email)
    if err := r.client.Set(ctx, key, "1", ttl).Err(); err != nil {
        return fmt.Errorf("redis mark email verified: %w", err)
    }
    return nil
}

func (r *redisStore) ConsumeEmailVerified(ctx context.Context, email string) (bool, error) {
    key := emailVerifiedKey(email)
    _, err := r.client.GetDel(ctx, key).Result()
    if err == redis.Nil {
        return false, nil
    }
    if err != nil {
        return false, fmt.Errorf("redis consume email verified: %w", err)
    }
    return true, nil
}

func emailVerifiedKey(email string) string {
    return fmt.Sprintf("auth:otp:register:%s:verified", email)
}


func (r *redisStore) ResetOTPAttempts(ctx context.Context, email string) error {
    key := otpAttemptsKey(email)
    if err := r.client.Del(ctx, key).Err(); err != nil {
        return fmt.Errorf("redis delete otp attempts: %w", err)
    }
    return nil
}

// ── Password Reset OTP ─────────────────────────────────────────

func (r *redisStore) SetPasswordResetOTP(ctx context.Context, email, otpHash string, ttl time.Duration) error {
    key := passwordResetOTPKey(email)
    if err := r.client.Set(ctx, key, otpHash, ttl).Err(); err != nil {
        return fmt.Errorf("redis set password reset otp: %w", err)
    }
    return nil
}

func (r *redisStore) GetPasswordResetOTP(ctx context.Context, email string) (string, bool, error) {
    key := passwordResetOTPKey(email)
    hash, err := r.client.Get(ctx, key).Result()
    if err == redis.Nil {
        return "", false, nil
    }
    if err != nil {
        return "", false, fmt.Errorf("redis get password reset otp: %w", err)
    }
    return hash, true, nil
}

func (r *redisStore) DeletePasswordResetOTP(ctx context.Context, email string) error {
    key := passwordResetOTPKey(email)
    if err := r.client.Del(ctx, key).Err(); err != nil {
        return fmt.Errorf("redis delete password reset otp: %w", err)
    }
    return nil
}

func (r *redisStore) IncrementPasswordResetOTPAttempts(ctx context.Context, email string, ttl time.Duration) (int64, error) {
    key := passwordResetOTPAttemptsKey(email)
    count, err := r.client.Incr(ctx, key).Result()
    if err != nil {
        return 0, fmt.Errorf("redis increment password reset otp attempts: %w", err)
    }
    if count == 1 {
        if err := r.client.Expire(ctx, key, ttl).Err(); err != nil {
            return count, fmt.Errorf("redis set password reset otp attempts ttl: %w", err)
        }
    }
    return count, nil
}

func (r *redisStore) TrySetPasswordResetResendCooldown(ctx context.Context, email string, ttl time.Duration) (bool, time.Duration, error) {
    key := passwordResetOTPCooldownKey(email)
    ok, err := r.client.SetNX(ctx, key, "1", ttl).Result()
    if err != nil {
        return false, 0, fmt.Errorf("redis set password reset cooldown: %w", err)
    }
    if ok {
        return true, ttl, nil
    }
    remainingTTL, err := r.client.TTL(ctx, key).Result()
    if err != nil {
        return false, 0, fmt.Errorf("redis get password reset cooldown ttl: %w", err)
    }
    return false, remainingTTL, nil
}

func (r *redisStore) ResetPasswordResetOTPAttempts(ctx context.Context, email string) error {
    key := passwordResetOTPAttemptsKey(email)
    if err := r.client.Del(ctx, key).Err(); err != nil {
        return fmt.Errorf("redis delete password reset otp attempts: %w", err)
    }
    return nil
}

// ── Key helpers ────────────────────────────────────────────────

func otpKey(email string) string {
    return fmt.Sprintf("auth:otp:register:%s", email)
}

func otpAttemptsKey(email string) string {
    return fmt.Sprintf("auth:otp:register:%s:attempts", email)
}

func otpCooldownKey(email string) string {
    return fmt.Sprintf("auth:otp:register:%s:cooldown", email)
}

func passwordResetOTPKey(email string) string {
    return fmt.Sprintf("auth:otp:password_reset:%s", email)
}

func passwordResetOTPAttemptsKey(email string) string {
    return fmt.Sprintf("auth:otp:password_reset:%s:attempts", email)
}

func passwordResetOTPCooldownKey(email string) string {
    return fmt.Sprintf("auth:otp:password_reset:%s:cooldown", email)
}