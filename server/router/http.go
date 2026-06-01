package main

import (
	"crypto"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
)

func startHTTPServer() {
	key, err := rsa.GenerateKey(cryptorand.Reader, 2048)
	if err != nil {
		log.Fatalf("[HTTP] failed to generate RSA key: %v", err)
	}
	log.Printf("[HTTP] RS256 key ready")
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.GET("/api/token/powersync", handleToken(key))
	r.GET("/api/.well-known/jwks.json", handleJWKS(key))
	r.POST("/api/embed", handleEmbed())
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
			SetHeader("Content-Type", "application/json").
			SetAuthToken(os.Getenv("HF_TOKEN"))
		resp, err := client.R().
			SetBody(map[string][]string{"inputs": {req.Text}}).
			Post("https://router.huggingface.co/hf-inference/models/BAAI/bge-large-zh-v1.5/pipeline/feature-extraction")
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
