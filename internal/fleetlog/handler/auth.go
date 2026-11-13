package handler

import (
	"context"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc/metadata"

	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/entity"
	"github.com/BwCloudWeGo/bw-cli/internal/fleetlog/service"
	apperrors "github.com/BwCloudWeGo/bw-cli/pkg/errors"
)

const (
	metadataAuthorization = "authorization"
	metadataVehicleToken  = "x-vehicle-token"
)

type sessionClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type authHelper struct {
	svc    *service.Service
	secret string
	issuer string
}

func newAuthHelper(svc *service.Service, secret string, issuer string) *authHelper {
	return &authHelper{svc: svc, secret: secret, issuer: issuer}
}

// actor 从签名令牌还原操作人，身份不来自请求体。
func (a *authHelper) actor(ctx context.Context) (entity.Actor, error) {
	if strings.TrimSpace(a.secret) == "" {
		return entity.Actor{}, apperrors.Unauthorized("fleetlog_auth_misconfigured", "未配置 fleetlog 会话签名密钥")
	}
	tokenText := bearerToken(metadataValue(ctx, metadataAuthorization))
	if tokenText == "" {
		return entity.Actor{}, apperrors.Unauthorized("fleetlog_missing_token", "缺少会话令牌")
	}
	claims := &sessionClaims{}
	parsed, err := jwt.ParseWithClaims(tokenText, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return []byte(a.secret), nil
	}, jwt.WithIssuer(a.issuer))
	if err != nil || !parsed.Valid {
		return entity.Actor{}, apperrors.Unauthorized("fleetlog_invalid_token", "会话令牌无效或已过期")
	}
	return a.svc.Actor(claims.UserID, claims.Role), nil
}

// vehicle 把车辆令牌映射为车辆编号。
func (a *authHelper) vehicle(ctx context.Context) (string, error) {
	token := strings.TrimSpace(metadataValue(ctx, metadataVehicleToken))
	if token == "" {
		return "", apperrors.Unauthorized("fleetlog_missing_vehicle_token", "缺少车辆凭证")
	}
	vehicleID, ok := a.svc.VehicleByToken(token)
	if !ok {
		return "", apperrors.Unauthorized("fleetlog_invalid_vehicle_token", "车辆凭证无效")
	}
	return vehicleID, nil
}

func metadataValue(ctx context.Context, key string) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(key); len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

func bearerToken(header string) string {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}
