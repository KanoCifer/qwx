package qwx

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Auth 是和风天气的认证方式。
//
// 官方支持 JWT 与 API KEY 两种，但**不能同时使用**（同时使用会导致认证失败）。
// 实现见 [NewJWTAuth] 与 [NewAPIKeyAuth]。
type Auth interface {
	// Apply 把认证信息写入请求。
	Apply(req *http.Request) error
}

// JWTAuth 用 Ed25519 私钥签发和风天气要求的 EdDSA JWT。
//
// 官方推荐 JWT 作为首选认证方式。签发的 token 有效期 15 分钟，
// 本实现会在过期前自动重新签名，调用方无需关心。
type JWTAuth struct {
	priv ed25519.PrivateKey

	// kid 是凭据的 Key ID（控制台创建凭据后可见）。
	kid string
	// sub 是凭据的 Project ID。
	sub string
	// iss 是开发者的 Developer ID（payload 的签发者）。
	iss string

	// tokenTTL 是 token 的有效期，默认 15 分钟（官方口径）。
	tokenTTL time.Duration
	// clockSkew 是 iat 回退的秒数，默认 30 秒（官方口径）。
	clockSkew time.Duration

	now func() time.Time

	mu        sync.Mutex
	token     string
	tokenTill time.Time
}

// JWTConfig 配置 [JWTAuth]。
type JWTConfig struct {
	// PrivateKeyPEM 是 Ed25519 私钥的 PEM 内容（PKCS#8 格式）。
	//
	// 生成方式：
	//
	//	openssl genpkey -algorithm ED25519 -out ed25519-private.pem
	//
	// 也接受含 "\\n" 转义的字符串（环境变量常见写法）。
	PrivateKeyPEM string
	// KeyID 是凭据的 Key ID，对应 JWT header 的 kid。
	KeyID string
	// ProjectID 是凭据的 Project ID，对应 JWT payload 的 sub。
	ProjectID string
	// DeveloperID 是开发者的 Developer ID，对应 JWT payload 的 iss。
	//
	// 官方口径是必填：iss 是签发者，上游按它校验凭据归属。值在控制台
	// 「设置」页查看，形如 Q 开头的 10 位字母数字。SDK 只校验非空，
	// 不校验格式 —— 上游口径可能调整，写死格式容易误杀合法配置。
	DeveloperID string
	// TokenTTL 覆盖默认的 15 分钟有效期。
	TokenTTL time.Duration
}

// NewJWTAuth 构造 [JWTAuth]。
//
// 私钥、Key ID、Project ID、Developer ID 四项都必填，缺任一项返回
// [ErrInvalidConfig]。Developer ID 对应 payload 的 iss，官方口径下上游
// 按它校验凭据归属，缺失会导致 401。
func NewJWTAuth(cfg JWTConfig) (*JWTAuth, error) {
	if cfg.PrivateKeyPEM == "" {
		return nil, fmt.Errorf("%w: empty private key", ErrInvalidConfig)
	}
	if cfg.KeyID == "" {
		return nil, fmt.Errorf("%w: empty key ID", ErrInvalidConfig)
	}
	if cfg.ProjectID == "" {
		return nil, fmt.Errorf("%w: empty project ID", ErrInvalidConfig)
	}
	if cfg.DeveloperID == "" {
		return nil, fmt.Errorf("%w: empty developer ID", ErrInvalidConfig)
	}

	key, err := parseEd25519PrivateKey(cfg.PrivateKeyPEM)
	if err != nil {
		return nil, err
	}

	ttl := cfg.TokenTTL
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	return &JWTAuth{
		priv:      key,
		kid:       cfg.KeyID,
		sub:       cfg.ProjectID,
		iss:       cfg.DeveloperID,
		tokenTTL:  ttl,
		clockSkew: 30 * time.Second,
		now:       time.Now,
	}, nil
}

// parseEd25519PrivateKey 解析 PKCS#8 编码的 Ed25519 私钥。
func parseEd25519PrivateKey(pemStr string) (ed25519.PrivateKey, error) {
	pemStr = strings.ReplaceAll(pemStr, "\\n", "\n")
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("%w: invalid PEM block", ErrInvalidConfig)
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: parse PKCS#8 private key: %v", ErrInvalidConfig, err)
	}
	edKey, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("%w: private key is not Ed25519", ErrInvalidConfig)
	}
	return edKey, nil
}

// Apply 把 Bearer token 写入请求头。
func (a *JWTAuth) Apply(req *http.Request) error {
	token, err := a.Token()
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}

// Token 返回当前的 JWT，必要时重新签发。
//
// token 在过期前 1 分钟失效并重新签发，避免请求发出时刚好过期。
func (a *JWTAuth) Token() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := a.now()
	if a.token != "" && now.Before(a.tokenTill) {
		return a.token, nil
	}

	token, err := a.sign(now)
	if err != nil {
		return "", err
	}
	a.token = token
	// 提前 1 分钟换新，留出请求在途的余量。
	a.tokenTill = now.Add(a.tokenTTL - time.Minute)
	return token, nil
}

// sign 生成一个 EdDSA JWT。
//
// payload 字段与官方口径一致：iss 是签发者（Developer ID），sub 是凭据的
// 项目 ID，iat 回退 30 秒容忍时钟偏差，exp = iat + tokenTTL。
func (a *JWTAuth) sign(now time.Time) (string, error) {
	iat := now.Add(-a.clockSkew)
	exp := iat.Add(a.tokenTTL)

	header := map[string]string{"alg": "EdDSA", "kid": a.kid}
	payload := map[string]any{
		"iat": iat.Unix(),
		"exp": exp.Unix(),
		"sub": a.sub,
		"iss": a.iss,
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("%w: marshal jwt header: %v", ErrInvalidConfig, err)
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("%w: marshal jwt payload: %v", ErrInvalidConfig, err)
	}

	signingInput := base64url(headerJSON) + "." + base64url(payloadJSON)
	sig := ed25519.Sign(a.priv, []byte(signingInput))
	return signingInput + "." + base64url(sig), nil
}

// base64url 做 JWT 用的 base64url 编码（无 padding）。
func base64url(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

// APIKeyAuth 用 API KEY 认证。
//
// 官方已公告：从 2027 年 1 月 1 日起限制用 API KEY 认证的每日请求量，
// 且 SDK 5+ 不再支持 API KEY。新项目建议改用 [JWTAuth]。
type APIKeyAuth struct {
	key string
}

// NewAPIKeyAuth 构造 [APIKeyAuth]。
func NewAPIKeyAuth(key string) (*APIKeyAuth, error) {
	if key == "" {
		return nil, fmt.Errorf("%w: empty API key", ErrInvalidConfig)
	}
	return &APIKeyAuth{key: key}, nil
}

// Apply 把 API KEY 写入请求头。
func (a *APIKeyAuth) Apply(req *http.Request) error {
	req.Header.Set("X-QW-Api-Key", a.key)
	return nil
}

// 编译期断言：两种认证方式都实现了 Auth。
var (
	_ Auth = (*JWTAuth)(nil)
	_ Auth = (*APIKeyAuth)(nil)
)
