package main

import (
	"fmt"

	_ "net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

var r *gin.Engine

func main() {
	rc := connectredis()
	token := getToken(rc)
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	r.Run(":8080")
}
func connectredis() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	pong, err := client.Ping().Result()
	fmt.Println(pong, err)
	return client
}
func getToken(rc *redis.Client) string {
	val, err := rc.Get("TDX_Token").Result()
	if err == nil {
		r.GET("/", func(c *gin.Context) {
			c.JSON(200, gin.H{})
		})
	}
	return val
}
