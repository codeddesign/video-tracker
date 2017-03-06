package main

import (
	"encoding/base64"
	"expvar"
	"fmt"
	"github.com/caarlos0/env"
	"github.com/garyburd/redigo/redis"
	"github.com/getsentry/raven-go"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"strconv"
	"time"
)

type RedisCommand struct {
	Command   string
	Key       string
	Field     string
	Increment int
}

type config struct {
	RedisPoolMaxIdle   int           `env:"REDIS_POOL_MAX_IDLE"`
	RedisPoolMaxActive int           `env:"REDIS_POOL_MAX_ACTIVE"`
	RedisPoolTimeout   time.Duration `env:"REDIS_POOL_TIMEOUT"`
	RedisConnection    string        `env:"REDIS_CONNECTION"`
	SentryDsn          string        `env:"SENTRY_DSN"`
	PipelineSize       int           `env:"PIPELINE_SIZE"`
}

const (
	servtimeout = time.Duration(15 * time.Second)
)

var (
	exp_events_processed = expvar.NewInt("events_processed")
)

var (
	exp_events_pipelined = expvar.NewInt("events_pipelined")
)

var (
	exp_events_sent = expvar.NewInt("events_sent")
)

func getConfig() *config {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	cfg := config{}
	errParse := env.Parse(&cfg)
	if errParse != nil {
		fmt.Printf("%+v\n", err)
	}
	return &cfg
}

func newPool() *redis.Pool {
	return &redis.Pool{
		MaxIdle:     cfg.RedisPoolMaxIdle,
		MaxActive:   cfg.RedisPoolMaxActive,
		Wait:        true,
		IdleTimeout: cfg.RedisPoolTimeout * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.DialURL(cfg.RedisConnection)
			if err != nil {
				raven.CaptureError(err, nil)
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
}

var cfg = getConfig()
var pool = newPool()
var pipeline = make(chan RedisCommand)

func handleTrackRequest(w http.ResponseWriter, r *http.Request) {
	campaign := r.URL.Query().Get("campaign")
	source := r.URL.Query().Get("source")
	status := r.URL.Query().Get("status")
	//rd := r.URL.Query().Get("_rd")
	platform := r.URL.Query().Get("platform")
	website := r.URL.Query().Get("w")
	tag := r.URL.Query().Get("tag")

	campaignRequiredParams := campaign != "" && source != "" && status != "" // && rd != ""
	analyticsRequiredParams := source == "visit" && platform != "" && website != ""
	backupRequiredParams := source == "backup" && campaign != "" && platform != ""

	// Required parameters
	if !campaignRequiredParams && !analyticsRequiredParams && !backupRequiredParams {
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

	if backupRequiredParams {
		//saveBackupToRedis(campaign, website)
	}

	exp_events_processed.Add(1)
	imageResponse(w)
}

func saveAnalyticsToRedis(website string, platform string) {
	if website == "" {
		return
	}

	value := "platform:" + platform

	pipeline <- RedisCommand{"HINCRBY", "website:" + website, value, 1}
}

func saveBackupToRedis(campaign string, website string) {
	value := "source:backup"

	if website != "" {
		value += ":website:" + website
	}

	pipeline <- RedisCommand{"HINCRBY", "campaign:" + campaign, value, 1}
}

func saveCampaignToRedis(source string, campaign string, tag string, status string, website string) {
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
			pipeline <- RedisCommand{"HINCRBY", "tag_requests", tag, 1}
		}
	}

	if website != "" {
		value += ":website:" + website
	}

	pipeline <- RedisCommand{"HINCRBY", "campaign:" + campaign, value, 1}
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

func redisPipeline(out chan RedisCommand) {
	commands := make([]RedisCommand, 0)

	for x := range out {
		commands = append(commands, x)
		exp_events_pipelined.Add(1)
		if len(commands) >= cfg.PipelineSize {
			go processRedisPipeline(commands)
			commands = commands[:0]
		}
	}
}

func processRedisPipeline(commands []RedisCommand) {
	c := pool.Get()
	defer c.Close()

	c.Do("MULTI")

	for _, command := range commands {
		c.Do(command.Command, command.Key, command.Field, command.Increment)
	}

	exp_events_sent.Add(1)
	c.Do("EXEC")
}

func main() {
	fmt.Println("Starting Go tracker...")
	fmt.Printf("%+v\n", cfg)

	go redisPipeline(pipeline)

	raven.SetDSN(cfg.SentryDsn)

	http.Handle("/track", http.TimeoutHandler(http.HandlerFunc(handleTrackRequest), servtimeout, ""))

	srv := &http.Server{
		Addr:           ":5000",
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	srv.ListenAndServe()
}
