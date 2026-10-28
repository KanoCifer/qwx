package qwx

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

// testPrivateKeyPEM 生成一对 Ed25519 密钥，返回 PKCS#8 私钥 PEM 与公钥。
func testPrivateKeyPEM(t *testing.T) (privPEM string, pub ed25519.PublicKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	block := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	return string(block), pub
}

// decodeJWTPart 解码 JWT 的某一段。
func decodeJWTPart(t *testing.T, token string, index int) []byte {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token has %d parts, want 3", len(parts))
	}
	data, err := base64.RawURLEncoding.DecodeString(parts[index])
	if err != nil {
		t.Fatalf("decode part %d: %v", index, err)
	}
	return data
}

func TestJWTAuthSignAndVerify(t *testing.T) {
	privPEM, pub := testPrivateKeyPEM(t)
	auth, err := NewJWTAuth(JWTConfig{
		PrivateKeyPEM: privPEM,
		KeyID:         "CKPM9U7FMW",
		ProjectID:     "2CTM8FQW28",
		DeveloperID:   "DEV123",
	})
	if err != nil {
		t.Fatalf("NewJWTAuth: %v", err)
	}

	now := time.Now()
	token, err := auth.Token()
	if err != nil {
		t.Fatalf("Token: %v", err)
	}

	// header：alg 与 kid。
	var header map[string]string
	if err := json.Unmarshal(decodeJWTPart(t, token, 0), &header); err != nil {
		t.Fatalf("decode header: %v", err)
	}
	if header["alg"] != "EdDSA" {
		t.Errorf("alg = %q, want EdDSA", header["alg"])
	}
	if header["kid"] != "CKPM9U7FMW" {
		t.Errorf("kid = %q, want CKPM9U7FMW", header["kid"])
	}

	// payload：iss / sub / iat / exp。
	var payload map[string]any
	if err := json.Unmarshal(decodeJWTPart(t, token, 1), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload["sub"] != "2CTM8FQW28" {
		t.Errorf("sub = %v, want 2CTM8FQW28", payload["sub"])
	}
	if payload["iss"] != "DEV123" {
		t.Errorf("iss = %v, want DEV123", payload["iss"])
	}

	iat, ok := payload["iat"].(float64)
	if !ok {
		t.Fatalf("iat is not a number: %v", payload["iat"])
	}
	exp, ok := payload["exp"].(float64)
	if !ok {
		t.Fatalf("exp is not a number: %v", payload["exp"])
	}
	// iat 回退 30 秒容忍时钟偏差。
	if want := float64(now.Unix() - 30); abs(iat-want) > 2 {
		t.Errorf("iat = %v, want ~%v", iat, want)
	}
	// exp = iat + 15 分钟。
	if want := iat + 900; abs(exp-want) > 2 {
		t.Errorf("exp = %v, want ~%v", exp, want)
	}

	// 签名必须能被公钥验证。
	if !ed25519.Verify(pub, []byte(strings.Join(strings.Split(token, ".")[:2], ".")), decodeJWTPart(t, token, 2)) {
		t.Error("signature does not verify against the public key")
	}
}

func TestJWTAuthCachesToken(t *testing.T) {
	privPEM, _ := testPrivateKeyPEM(t)
	auth, err := NewJWTAuth(JWTConfig{
		PrivateKeyPEM: privPEM,
		KeyID:         "K",
		ProjectID:     "P",
	})
	if err != nil {
		t.Fatalf("NewJWTAuth: %v", err)
	}

	first, err := auth.Token()
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	second, err := auth.Token()
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if first != second {
		t.Error("token should be reused before expiry")
	}
}

func TestJWTAuthRefreshesExpiredToken(t *testing.T) {
	privPEM, _ := testPrivateKeyPEM(t)
	auth, err := NewJWTAuth(JWTConfig{
		PrivateKeyPEM: privPEM,
		KeyID:         "K",
		ProjectID:     "P",
		TokenTTL:      2 * time.Minute,
	})
	if err != nil {
		t.Fatalf("NewJWTAuth: %v", err)
	}

	now := time.Now()
	auth.now = func() time.Time { return now }
	first, err := auth.Token()
	if err != nil {
		t.Fatalf("Token: %v", err)
	}

	// 推进到过期之后，应重新签发。
	auth.now = func() time.Time { return now.Add(10 * time.Minute) }
	second, err := auth.Token()
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if first == second {
		t.Error("token should be refreshed after expiry")
	}
}

func TestJWTAuthRequiresConfig(t *testing.T) {
	privPEM, _ := testPrivateKeyPEM(t)

	cases := []JWTConfig{
		{KeyID: "K", ProjectID: "P"},
		{PrivateKeyPEM: privPEM, ProjectID: "P"},
		{PrivateKeyPEM: privPEM, KeyID: "K"},
	}
	for i, cfg := range cases {
		if _, err := NewJWTAuth(cfg); err == nil {
			t.Errorf("case %d: expected error", i)
		} else if !errors.Is(err, ErrInvalidConfig) {
			t.Errorf("case %d: error = %v, want ErrInvalidConfig", i, err)
		}
	}
}

func TestJWTAuthRejectsBadKey(t *testing.T) {
	cases := []string{
		"not a pem",
		"-----BEGIN PRIVATE KEY-----\nnotbase64\n-----END PRIVATE KEY-----\n",
	}
	for _, pemStr := range cases {
		if _, err := NewJWTAuth(JWTConfig{
			PrivateKeyPEM: pemStr,
			KeyID:         "K",
			ProjectID:     "P",
		}); err == nil {
			t.Errorf("pem %q: expected error", pemStr)
		}
	}
}

func TestJWTAuthAcceptsEscapedNewlines(t *testing.T) {
	privPEM, _ := testPrivateKeyPEM(t)
	escaped := strings.ReplaceAll(privPEM, "\n", "\\n")

	// 环境变量里常写成转义换行，应能正常解析。
	if _, err := NewJWTAuth(JWTConfig{
		PrivateKeyPEM: escaped,
		KeyID:         "K",
		ProjectID:     "P",
	}); err != nil {
		t.Fatalf("NewJWTAuth with escaped newlines: %v", err)
	}
}

func TestJWTAuthAppliesHeader(t *testing.T) {
	privPEM, _ := testPrivateKeyPEM(t)
	auth, err := NewJWTAuth(JWTConfig{
		PrivateKeyPEM: privPEM,
		KeyID:         "K",
		ProjectID:     "P",
	})
	if err != nil {
		t.Fatalf("NewJWTAuth: %v", err)
	}

	req, err := newTestRequest()
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if err := auth.Apply(req); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got := req.Header.Get("Authorization"); !strings.HasPrefix(got, "Bearer ") {
		t.Errorf("Authorization = %q, want Bearer prefix", got)
	}
}

func TestAPIKeyAuth(t *testing.T) {
	auth, err := NewAPIKeyAuth("ABCD1234EFGH")
	if err != nil {
		t.Fatalf("NewAPIKeyAuth: %v", err)
	}
	req, err := newTestRequest()
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if err := auth.Apply(req); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got := req.Header.Get("X-QW-Api-Key"); got != "ABCD1234EFGH" {
		t.Errorf("X-QW-Api-Key = %q", got)
	}
	// API KEY 与 JWT 不能同时使用。
	if req.Header.Get("Authorization") != "" {
		t.Error("API key auth must not set Authorization header")
	}
}

func TestNewAPIKeyAuthRejectsEmpty(t *testing.T) {
	if _, err := NewAPIKeyAuth(""); err == nil {
		t.Error("expected error")
	}
}

func TestNewClientRequiresAuth(t *testing.T) {
	if _, err := NewClient(nil); err == nil {
		t.Error("expected error")
	} else if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("error = %v, want ErrInvalidConfig", err)
	}
}

func TestNewClientNormalizesBaseURL(t *testing.T) {
	c, err := NewClient(&APIKeyAuth{key: "k"}, WithBaseURL("abcxyz.qweatherapi.com"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.baseURL != "https://abcxyz.qweatherapi.com" {
		t.Errorf("baseURL = %q", c.baseURL)
	}

	// 尾部斜杠要去掉，否则拼出 "//weather/..."。
	c, err = NewClient(&APIKeyAuth{key: "k"}, WithBaseURL("https://abcxyz.qweatherapi.com/"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.baseURL != "https://abcxyz.qweatherapi.com" {
		t.Errorf("baseURL = %q", c.baseURL)
	}
}

func TestParseLocation(t *testing.T) {
	t.Run("coordinate pair", func(t *testing.T) {
		loc, err := ParseLocation("116.41,39.92")
		if err != nil {
			t.Fatalf("ParseLocation: %v", err)
		}
		if loc.Lat != 39.92 || loc.Lon != 116.41 {
			t.Errorf("lat/lon = %v/%v", loc.Lat, loc.Lon)
		}
		// v7/GeoAPI 的 location 参数是"经度,纬度"。
		if got := loc.Query(); got != "116.41,39.92" {
			t.Errorf("Query = %q, want 116.41,39.92", got)
		}
		// 新一代接口的路径是"纬度,经度"。
		lat, lon, err := loc.Path()
		if err != nil {
			t.Fatalf("Path: %v", err)
		}
		if lat != "39.92" || lon != "116.41" {
			t.Errorf("Path = %q/%q, want 39.92/116.41", lat, lon)
		}
	})

	t.Run("location id", func(t *testing.T) {
		loc, err := ParseLocation("101010100")
		if err != nil {
			t.Fatalf("ParseLocation: %v", err)
		}
		if loc.ID != "101010100" {
			t.Errorf("ID = %q", loc.ID)
		}
		if loc.HasCoordinates() {
			t.Error("LocationID should not have coordinates")
		}
		if got := loc.Query(); got != "101010100" {
			t.Errorf("Query = %q", got)
		}
	})

	t.Run("poi id with letter suffix", func(t *testing.T) {
		loc, err := ParseLocation("10101010012A")
		if err != nil {
			t.Fatalf("ParseLocation: %v", err)
		}
		if loc.ID != "10101010012A" {
			t.Errorf("ID = %q", loc.ID)
		}
	})

	t.Run("tide station id", func(t *testing.T) {
		loc, err := ParseLocation("P2352")
		if err != nil {
			t.Fatalf("ParseLocation: %v", err)
		}
		if loc.ID != "P2352" {
			t.Errorf("ID = %q", loc.ID)
		}
	})

	t.Run("bare longitude is rejected", func(t *testing.T) {
		// 裸经度三种形态都不沾，上游直接 400。
		if _, err := ParseLocation("121.47"); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("out of range", func(t *testing.T) {
		for _, s := range []string{"181,39.92", "116.41,91", "200,100"} {
			if _, err := ParseLocation(s); err == nil {
				t.Errorf("%q: expected error", s)
			}
		}
	})

	t.Run("NaN and Inf are rejected", func(t *testing.T) {
		for _, s := range []string{"NaN,39.92", "116.41,NaN", "Inf,0", "0,Inf"} {
			if _, err := ParseLocation(s); err == nil {
				t.Errorf("%q: expected error", s)
			}
		}
	})

	t.Run("empty", func(t *testing.T) {
		if _, err := ParseLocation("   "); err == nil {
			t.Error("expected error")
		}
	})
}

func TestLocationPathRejectsMissingCoordinates(t *testing.T) {
	loc := Location{ID: "101010100"}
	if _, _, err := loc.Path(); err == nil {
		t.Error("expected error")
	}
}

func TestLocationQueryPrefersCoordinates(t *testing.T) {
	loc := Location{ID: "101010100", Lat: 39.92, Lon: 116.41}
	if got := loc.Query(); got != "116.41,39.92" {
		t.Errorf("Query = %q, want coordinates", got)
	}
}

func TestMemoryCache(t *testing.T) {
	c := NewMemoryCache()
	ctx := context.Background()

	if _, ok, err := c.Get(ctx, "missing"); err != nil || ok {
		t.Errorf("missing key: ok=%v err=%v", ok, err)
	}

	if err := c.Set(ctx, "k", []byte("v"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	data, ok, err := c.Get(ctx, "k")
	if err != nil || !ok || string(data) != "v" {
		t.Errorf("Get = %q, %v, %v", data, ok, err)
	}

	// 过期后应当 miss 并清理条目。
	c.now = func() time.Time { return time.Now().Add(2 * time.Minute) }
	if _, ok, _ := c.Get(ctx, "k"); ok {
		t.Error("expired key should miss")
	}
	if len(c.entries) != 0 {
		t.Errorf("entries = %d, want 0 after expiry", len(c.entries))
	}
}

// abs 返回浮点数的绝对值。
func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// newTestRequest 造一个用于测试认证头的请求。
func newTestRequest() (*http.Request, error) {
	return http.NewRequest(http.MethodGet, "https://example.com/", nil)
}
