package main

import (
	"encoding/base64"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"strconv"
)

func newPool() *redis.Pool {
	return &redis.Pool {
		MaxIdle:   50,
		MaxActive: 10000,
		Dial: func() (redis.Conn, error) {
			c, err := redis.DialURL(os.Getenv("REDIS_CONNECTION"))
			if err != nil {
				panic(err.Error())
			}
			return c, err
		},
	}
}

var pool = newPool()

func handleTrackRequest(w http.ResponseWriter, r *http.Request) {
	campaign := r.URL.Query().Get("campaign")
	source := r.URL.Query().Get("source")
	status := r.URL.Query().Get("status")
	//rd := r.URL.Query().Get("_rd")
	platform := r.URL.Query().Get("platform")
	website := r.URL.Query().Get("w")
	tag := r.URL.Query().Get("tag")

	campaignRequiredParams := campaign != "" && source != "" && status != ""// && rd != ""
	analyticsRequiredParams := source == "visit" && platform != "" && website != ""

	// Required parameters
	if !campaignRequiredParams && !analyticsRequiredParams {
		return
	}

	if _, err := strconv.Atoi(website); err != nil {
		website = ""
	}

	if website == "0" {
		website = ""
	}

	if campaignRequiredParams {
		saveCampaignToRedis(source, campaign, tag, status, website)
	}

	if analyticsRequiredParams {
		saveAnalyticsToRedis(website, platform)
	}

	imageResponse(w)
}

func saveAnalyticsToRedis(website string, platform string) {
	c := pool.Get()
	defer c.Close()

	if website == "" {
		return
	}

	value := "platform:" + platform

	c.Do("HINCRBY", "website:"+website, value, 1)
}

func saveCampaignToRedis(source string, campaign string, tag string, status string, website string) {
	c := pool.Get()
	defer c.Close()

	if tag == "false" {
		tag = ""
	}

	if _, err := strconv.Atoi(status); err != nil {
		status = "901"
	}

	value := "source:" + source + ":status:" + status

	if tag != "" {
		value += ":tag:" + tag

		if source == "tag" {
			c.Do("HINCRBY", "tag_requests", tag, 1)
		}
	}

	if website != "" {
		value += ":website:" + website
	}

	c.Do("HINCRBY", "campaign:"+campaign, value, 1)
}

func imageResponse(w http.ResponseWriter) {
	imageBase64 := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABAQMAAAAl21bKAAAAA1BMVEUAAACnej3aAAAAAXRSTlMAQObYZgAAAApJREFUCNdjYAAAAAIAAeIhvDMAAAAASUVORK5CYII="

	image, err := base64.StdEncoding.DecodeString(imageBase64)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	fmt.Fprintf(w, "%s", image)
}

func main() {
	fmt.Println("Starting Go tracker...")

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	http.HandleFunc("/track", handleTrackRequest)
	http.ListenAndServe(":5000", nil)
}
