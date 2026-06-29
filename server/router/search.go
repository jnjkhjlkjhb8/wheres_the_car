package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type searchResult struct {
	Type   string   `json:"type"`
	UID    string   `json:"uid"`
	Name   string   `json:"name"`
	City   string   `json:"city"`
	Depart string   `json:"depart"`
	Destin string   `json:"destin"`
	Lat    *float64 `json:"lat"`
	Lon    *float64 `json:"lon"`
}

func toVecLiteral(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = strconv.FormatFloat(float64(f), 'f', -1, 32)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func embedQuery(text string) ([]float32, error) {
	client := resty.New().SetHeader("Content-Type", "application/json")
	resp, err := client.R().
		SetBody(map[string]interface{}{
			"model": "qwen3-embedding:0.6b",
			"input": []string{text},
		}).
		Post("http://ollama:11434/api/embed")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("ollama %d: %s", resp.StatusCode(), resp.Body())
	}
	var result struct {
		Embeddings [][]float32 `json:"embeddings"`
	}
	if err := json.Unmarshal(resp.Body(), &result); err != nil || len(result.Embeddings) == 0 {
		return nil, fmt.Errorf("embed parse: %v", err)
	}
	return result.Embeddings[0], nil
}

func vectorSearch(ctx context.Context, q string, limit int, db *pgxpool.Pool) ([]searchResult, error) {
	vec, embedErr := embedQuery(q)
	if embedErr != nil {
		log.Printf("[SEARCH] embed failed, using text fallback: %v", embedErr)
		return textFallback(ctx, q, limit, db)
	}
	rows, err := db.Query(ctx, `
		SELECT type, uid, name, city, depart, destin,
		       ST_Y(geom), ST_X(geom)
		FROM search_vector
		ORDER BY embedding <=> $1::vector
		LIMIT $2`,
		toVecLiteral(vec), limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []searchResult
	for rows.Next() {
		var r searchResult
		if err := rows.Scan(&r.Type, &r.UID, &r.Name, &r.City, &r.Depart, &r.Destin, &r.Lat, &r.Lon); err == nil {
			results = append(results, r)
		}
	}
	return results, nil
}

func textFallback(ctx context.Context, q string, limit int, db *pgxpool.Pool) ([]searchResult, error) {
	rows, err := db.Query(ctx, `
		SELECT type, uid, name, city, depart, destin,
		       ST_Y(geom), ST_X(geom)
		FROM search_vector
		WHERE name % $1
		   OR name  ILIKE '%' || $1 || '%'
		   OR depart ILIKE '%' || $1 || '%'
		   OR destin ILIKE '%' || $1 || '%'
		ORDER BY similarity(name, $1) DESC
		LIMIT $2`,
		q, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []searchResult
	for rows.Next() {
		var r searchResult
		if err := rows.Scan(&r.Type, &r.UID, &r.Name, &r.City, &r.Depart, &r.Destin, &r.Lat, &r.Lon); err == nil {
			results = append(results, r)
		}
	}
	return results, nil
}

func expandStationRoutes(ctx context.Context, primary []searchResult, db *pgxpool.Pool) []searchResult {
	var stationNames []string
	seen := make(map[string]bool)
	for _, r := range primary {
		seen[r.UID] = true
		if r.Type == "bus_station" {
			stationNames = append(stationNames, r.Name)
		}
	}
	if len(stationNames) == 0 {
		return primary
	}
	rows, err := db.Query(ctx, `
		SELECT sv.type, sv.uid, sv.name, sv.city, sv.depart, sv.destin,
		       ST_Y(sv.geom), ST_X(sv.geom)
		FROM bus_station_stop_map bssm
		JOIN search_vector sv
		  ON sv.uid = bssm.sub_route_uid AND sv.type = 'bus_route'
		WHERE bssm.station_name = ANY($1)`,
		stationNames,
	)
	if err != nil {
		log.Printf("[SEARCH] route expansion error: %v", err)
		return primary
	}
	defer rows.Close()
	var extra []searchResult
	for rows.Next() {
		var r searchResult
		if err := rows.Scan(&r.Type, &r.UID, &r.Name, &r.City, &r.Depart, &r.Destin, &r.Lat, &r.Lon); err == nil {
			if !seen[r.UID] {
				extra = append(extra, r)
				seen[r.UID] = true
			}
		}
	}
	return append(primary, extra...)
}

func isNumericQuery(q string) bool {
	if len(q) == 0 {
		return false
	}
	for _, c := range q {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func trainNumberSearch(ctx context.Context, q string, db *pgxpool.Pool) []searchResult {
	rows, err := db.Query(ctx, `
		SELECT type, uid, name, city, depart, destin,
		       ST_Y(geom), ST_X(geom)
		FROM search_vector
		WHERE uid = $1 AND type IN ('tra_train', 'thsr_train')`,
		q,
	)
	if err != nil {
		log.Printf("[SEARCH] train number search error: %v", err)
		return nil
	}
	defer rows.Close()
	var results []searchResult
	for rows.Next() {
		var r searchResult
		if err := rows.Scan(&r.Type, &r.UID, &r.Name, &r.City, &r.Depart, &r.Destin, &r.Lat, &r.Lon); err == nil {
			results = append(results, r)
		}
	}
	return results
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
		var trainResults []searchResult
		if isNumericQuery(q) {
			trainResults = trainNumberSearch(ctx, q, db)
		}
		results, err := vectorSearch(ctx, q, limit, db)
		if err != nil {
			log.Printf("[SEARCH] error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
			return
		}
		results = expandStationRoutes(ctx, results, db)
		if len(trainResults) > 0 {
			seen := make(map[string]bool)
			for _, r := range trainResults {
				seen[r.UID+r.Type] = true
			}
			for _, r := range results {
				if !seen[r.UID+r.Type] {
					trainResults = append(trainResults, r)
				}
			}
			results = trainResults
		}
		c.JSON(http.StatusOK, gin.H{"results": results})
	}
}
