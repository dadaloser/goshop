package v1

import (
	"context"
	"net"
	"strings"

	"goshop/app/pkg/code"
	"goshop/pkg/errors"
	"goshop/pkg/log"
)

func (us *userService) ensurePasswordLoginAllowed(ctx context.Context, username, clientIP string) error {
	if us == nil || us.loginAttempts == nil {
		return us.ensurePasswordLoginIPAllowed(ctx, clientIP)
	}

	locked, err := us.loginAttempts.IsLocked(ctx, username)
	if err != nil {
		log.Errorf("check password login attempts failed: %v", err)
		return errors.WithCode(code.ErrUserLoginLocked, "登录暂时不可用，请稍后重试")
	}
	if locked {
		return errors.WithCode(code.ErrUserLoginLocked, "登录失败次数过多，请稍后重试")
	}
	return us.ensurePasswordLoginIPAllowed(ctx, clientIP)
}

func (us *userService) ensurePasswordLoginIPAllowed(ctx context.Context, clientIP string) error {
	if strings.TrimSpace(clientIP) == "" || us == nil || us.loginIPAttempts == nil {
		return nil
	}

	locked, err := us.loginIPAttempts.IsLocked(ctx, clientIP)
	if err != nil {
		log.Errorf("check password login attempts by ip failed: %v", err)
		return errors.WithCode(code.ErrUserLoginLocked, "登录暂时不可用，请稍后重试")
	}
	if locked {
		return errors.WithCode(code.ErrUserLoginLocked, "登录失败次数过多，请稍后重试")
	}
	return nil
}

func (us *userService) recordPasswordLoginFailure(ctx context.Context, username, clientIP string) error {
	if us == nil {
		return nil
	}

	locked := false
	if us.loginAttempts != nil {
		accountLocked, err := us.loginAttempts.RecordFailure(ctx, username)
		if err != nil {
			log.Errorf("record password login failure failed: %v", err)
			return errors.WithCode(code.ErrUserLoginLocked, "登录暂时不可用，请稍后重试")
		}
		locked = locked || accountLocked
	}
	if strings.TrimSpace(clientIP) != "" && us.loginIPAttempts != nil {
		ipLocked, err := us.loginIPAttempts.RecordFailure(ctx, clientIP)
		if err != nil {
			log.Errorf("record password login failure by ip failed: %v", err)
			return errors.WithCode(code.ErrUserLoginLocked, "登录暂时不可用，请稍后重试")
		}
		locked = locked || ipLocked
	}
	if locked {
		return errors.WithCode(code.ErrUserLoginLocked, "登录失败次数过多，请稍后重试")
	}
	return nil
}

func (us *userService) resetPasswordLoginFailures(ctx context.Context, username, clientIP string) {
	if us == nil {
		return
	}

	if us.loginAttempts != nil {
		if err := us.loginAttempts.Reset(ctx, username); err != nil {
			log.Warnf("reset password login failures failed: %v", err)
		}
	}
	if strings.TrimSpace(clientIP) != "" && us.loginIPAttempts != nil {
		if err := us.loginIPAttempts.Reset(ctx, clientIP); err != nil {
			log.Warnf("reset password login failures by ip failed: %v", err)
		}
	}
}

func (us *userService) ensureSmsCodeAllowed(ctx context.Context, mobile string, codeType uint) error {
	if us == nil || us.smsAttempts == nil {
		return nil
	}

	locked, err := us.smsAttempts.IsLocked(ctx, mobile, codeType)
	if err != nil {
		log.Errorf("check sms verification attempts failed: %v", err)
		return errors.WithCode(code.ErrSmsVerifyLocked, "验证码验证暂时不可用，请稍后重试")
	}
	if locked {
		return errors.WithCode(code.ErrSmsVerifyLocked, "验证码错误次数过多，请稍后重试")
	}
	return nil
}

func (us *userService) recordSmsCodeFailure(ctx context.Context, mobile string, codeType uint) error {
	if us == nil || us.smsAttempts == nil {
		return nil
	}

	locked, err := us.smsAttempts.RecordFailure(ctx, mobile, codeType)
	if err != nil {
		log.Errorf("record sms verification failure failed: %v", err)
		return errors.WithCode(code.ErrSmsVerifyLocked, "验证码验证暂时不可用，请稍后重试")
	}
	if locked {
		return errors.WithCode(code.ErrSmsVerifyLocked, "验证码错误次数过多，请稍后重试")
	}
	return nil
}

func (us *userService) resetSmsCodeFailures(ctx context.Context, mobile string, codeType uint) {
	if us == nil || us.smsAttempts == nil {
		return
	}

	if err := us.smsAttempts.Reset(ctx, mobile, codeType); err != nil {
		log.Warnf("reset sms verification failures failed: %v", err)
	}
}

func normalizeLoginIdentifier(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func loginClientIP(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if provider, ok := ctx.(interface{ ClientIP() string }); ok {
		return normalizeClientIP(provider.ClientIP())
	}
	if provider, ok := ctx.(interface{ GetHeader(string) string }); ok {
		if forwarded := strings.TrimSpace(provider.GetHeader("X-Forwarded-For")); forwarded != "" {
			parts := strings.Split(forwarded, ",")
			if len(parts) > 0 {
				return normalizeClientIP(parts[0])
			}
		}
		return normalizeClientIP(provider.GetHeader("X-Real-IP"))
	}
	return ""
}

func normalizeClientIP(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(raw); err == nil {
		raw = host
	}
	if ip := net.ParseIP(raw); ip != nil {
		return ip.String()
	}
	return ""
}
