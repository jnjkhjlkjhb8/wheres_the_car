package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type searchResult struct {
	Type   string  `json:"type"`
	UID    string  `json:"uid"`
	Name   string  `json:"name"`
	City   string  `json:"city"`
	Depart string  `json:"depart"`
	Destin string  `json:"destin"`
	Score  float32 `json:"score"`
}

func handleSearch(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		q := c.Query("q")
		if len(q) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "q required"})
			return
		}
		limit := 20
		if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 && l <= 50 {
			limit = l
		}

		ctx := c.Request.Context()
		results, err := vectorSearch(ctx, q, limit, db)
		if err != nil {
			log.Printf("[SEARCH] error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"results": results})
	}
}
func vectorSearch(ctx context.Context, q string, limit int, db *pgxpool.Pool) ([]searchResult, error) {
	const maxCandidates = 300
	rows, err := db.Query(ctx, `
		SELECT type, uid, name, city, depart, destin, blob
		FROM search_vector
		WHERE name    ILIKE '%' || $1 || '%'
		   OR city    ILIKE '%' || $1 || '%'
		   OR depart  ILIKE '%' || $1 || '%'
		   OR destin  ILIKE '%' || $1 || '%'
		LIMIT $2`,
		q, maxCandidates,
	)
	if err != nil {
		return nil, err
	}
	type entry struct {
		r    searchResult
		blob []byte
	}
	var candidates []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.r.Type, &e.r.UID, &e.r.Name, &e.r.City, &e.r.Depart, &e.r.Destin, &e.blob); err == nil {
			candidates = append(candidates, e)
		}
	}
	rows.Close()
	qVec, embedErr := embedQuery(q)
	if embedErr != nil {
		log.Printf("[SEARCH] embed skipped (falling back to text): %v", embedErr)
	}
	results := make([]searchResult, 0, len(candidates))
	for _, e := range candidates {
		r := e.r
		if embedErr == nil && len(e.blob) == len(qVec)*4 {
			r.Score = cosine(qVec, unzip(e.blob))
		} else {
			r.Score = textScore(q, r.Name)
		}
		results = append(results, r)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}
func embedQuery(text string) ([]float32, error) {
	client := resty.New().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(os.Getenv("HF_TOKEN"))
	resp, err := client.R().
		SetBody(map[string][]string{"inputs": {text}}).
		Post("https://router.huggingface.co/hf-inference/models/BAAI/bge-large-zh-v1.5/pipeline/feature-extraction")
	if err != nil {
		return nil, err
	}
	var vecs [][]float32
	if err := json.Unmarshal(resp.Body(), &vecs); err != nil || len(vecs) == 0 {
		return nil, err
	}
	return vecs[0], nil
}
func unzip(blob []byte) []float32 {
	n := len(blob) / 4
	out := make([]float32, n)
	for i := range out {
		bits := binary.LittleEndian.Uint32(blob[i*4 : (i+1)*4])
		out[i] = math.Float32frombits(bits)
	}
	return out
}
func cosine(a, b []float32) float32 {
	var dot, normA, normB float64
	for i := range a {
		av, bv := float64(a[i]), float64(b[i])
		dot += av * bv
		normA += av * av
		normB += bv * bv
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return float32(dot / (math.Sqrt(normA) * math.Sqrt(normB)))
}
func textScore(query, name string) float32 {
	if len(name) == 0 {
		return 0
	}
	switch {
	case name == query:
		return 1.0
	case len(name) >= len(query) && name[:len(query)] == query:
		return 0.8
	default:
		return 0.5
	}
}
