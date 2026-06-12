package main

import (
	"crypto"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

func startHTTPServer(db *pgxpool.Pool) {
	key, err := loadOrGenerateKey()
	if err != nil {
		log.Fatalf("[HTTP] failed to load/generate RSA key: %v", err)
	}
	log.Printf("[HTTP] RS256 key ready")
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.GET("/api/token/powersync", handleToken(key))
	r.GET("/api/.well-known/jwks.json", handleJWKS(key))
	r.POST("/api/embed", handleEmbed())
	r.GET("/api/search", handleSearch(db))
	log.Printf("[HTTP] server running on 0.0.0.0:8080")
	if err := r.Run("0.0.0.0:8080"); err != nil {
		log.Fatalf("[HTTP] server failed: %v", err)
	}
}
func handleToken(key *rsa.PrivateKey) gin.HandlerFunc {
	return func(c *gin.Context) {
		now := time.Now()
		token, err := signRS256(key, map[string]any{
			"sub": "powersync-client",
			"aud": "powersync",
			"iat": now.Unix(),
			"exp": now.Add(24 * time.Hour).Unix(),
		})
		if err != nil {
			c.JSON(500, gin.H{"error": "sign failed"})
			return
		}
		c.JSON(200, gin.H{"token": token})
	}
}
func handleJWKS(key *rsa.PrivateKey) gin.HandlerFunc {
	pub := &key.PublicKey
	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())
	body := gin.H{
		"keys": []gin.H{{
			"kty": "RSA", "use": "sig", "alg": "RS256", "kid": "k1",
			"n": n, "e": e,
		}},
	}
	return func(c *gin.Context) { c.JSON(200, body) }
}
func handleEmbed() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Text string `json:"text" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		client := resty.New().
			SetHeader("Content-Type", "application/json")
		resp, err := client.R().
			SetBody(map[string][]string{"inputs": {req.Text}}).
			Post("http://embed:8082/embed")
		if err != nil {
			log.Printf("[HTTP] embed request failed: %v", err)
			c.JSON(500, gin.H{"error": "embed failed"})
			return
		}
		defer resp.RawResponse.Body.Close()
		var vecs [][]float32
		if err := json.Unmarshal(resp.Body(), &vecs); err != nil || len(vecs) == 0 {
			log.Printf("[HTTP] embed parse failed: %v body=%s", err, resp.Body())
			c.JSON(500, gin.H{"error": "parse failed"})
			return
		}
		out := make([]float64, len(vecs[0]))
		for i, f := range vecs[0] {
			out[i] = float64(f)
		}
		c.JSON(200, gin.H{"vector": out})
	}
}
func signRS256(key *rsa.PrivateKey, claims map[string]any) (string, error) {
	header := b64j(map[string]string{"alg": "RS256", "typ": "JWT", "kid": "k1"})
	payload := b64j(claims)
	msg := header + "." + payload
	h := sha256.New()
	h.Write([]byte(msg))
	sig, err := rsa.SignPKCS1v15(cryptorand.Reader, key, crypto.SHA256, h.Sum(nil))
	if err != nil {
		return "", err
	}
	return msg + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}
func b64j(v any) string {
	b, _ := json.Marshal(v)
	return base64.RawURLEncoding.EncodeToString(b)
}

func loadOrGenerateKey() (*rsa.PrivateKey, error) {
	const keyFile = "/tmp/powersync_key.pem"
	data, err := os.ReadFile(keyFile)
	if err == nil {
		block, _ := pem.Decode(data)
		if block != nil {
			key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err == nil {
				log.Printf("[HTTP] loaded persisted RSA key from %s", keyFile)
				return key, nil
			}
		}
	}
	key, err := rsa.GenerateKey(cryptorand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	data = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	_ = os.WriteFile(keyFile, data, 0600)
	log.Printf("[HTTP] generated new RSA key and persisted to %s", keyFile)
	return key, nil
}
