package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-co-op/gocron/v2"
)

const apiURL = "http://192.168.1.86:3000/api/refresh-category"

type Category struct {
	Name string
	ID   string
}

const (
	colorReset = "\033[0m"
	colorRed   = "\033[31m"
	colorGreen = "\033[32m"
	colorBlue  = "\033[34m"
)

func blue(s string) string  { return colorBlue + s + colorReset }
func green(s string) string { return colorGreen + s + colorReset }
func red(s string) string   { return colorRed + s + colorReset }

var categories = []Category{
	{"mobile",     "696f40673b0d1eb957a92f5b"},
	{"handsfree",  "6972014e47421019c5859a53"},
	{"powerbank",  "6986ee0a885b8bd8e7e902d8"},
	{"appliances", "69846a20b913b6a46b629340"},
	{"watch",      "696f37003b0d1eb957a92d48"},
	{"smarthome",  "69888b8746181a3ce235b69d"},
	{"speaker",    "696f40533b0d1eb957a92f44"},
	{"laptop",     "6974caba5f71415aaa31c384"},
	{"gamestore",  "6981c7d95569e42b38dd2847"},
	{"beauty",     "6984552e6e16d3cdbff792e4"},
}

var baseTimes = []struct{ Hour, Minute int }{
	{8, 0},
	{9, 0},
	{10, 0},
	{13, 0},
	{15, 0},
	{17, 0},
	{19, 0},
}

func main() {
	s, err := gocron.NewScheduler() // .WithLocation(time.UTC) if you want UTC
	if err != nil {
		log.Fatal(err)
	}

	for _, bt := range baseTimes {
		base := bt // capture

		_, err := s.NewJob(
			gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(uint(base.Hour), uint(base.Minute), 0))),
			gocron.NewTask(func() {
				log.Printf("Base time %02d:%02d started → processing %d categories sequentially", base.Hour, base.Minute, len(categories))

				for i, cat := range categories {
					success := sendRefresh(cat.Name, cat.ID)
					if success {
						// If this is not the last category, wait 5 min before next
						if i < len(categories)-1 {
							log.Printf("[%s] Success - waiting 5 minutes before next category", cat.Name)
							time.Sleep(5 * time.Minute)
						}
					} else {
						// On failure, log and continue to next without wait (or you could add retry logic here)
						log.Printf("[%s] Failed - continuing to next category without wait", cat.Name)
					}
				}

				log.Printf("Base time %02d:%02d completed", base.Hour, base.Minute)
			}),
		)
		if err != nil {
			log.Printf("Failed to schedule base %02d:%02d: %v", base.Hour, base.Minute, err)
		} else {
			log.Printf("Scheduled sequential processing at %02d:%02d", base.Hour, base.Minute)
		}
	}

	s.Start()

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		log.Println("Shutting down scheduler...")
		s.Shutdown()
		cancel()
	}()

	<-ctx.Done()
	log.Println("Exited.")
}

func sendRefresh(name, catID string) bool {
	log.Printf("[%s] Sending { \"category\": \"%s\" }", blue(name), catID)

	payload := map[string]string{"category": catID}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", apiURL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", "c6f1b9d3b4e84d7f9a6a2f0c7e1a8b5d4f2c9e7a6b8d0f1c3a5e9b7d2f4a") // ← add if needed

	client := &http.Client{Timeout: 10 * time.Minute}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[%s] %s",
			blue(name),
			red("FAILED: "+err.Error()),
		)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		log.Printf("[%s] %s",
			blue(name),
			green("SUCCESS → "+resp.Status),
		)
		return true
	} else {
		log.Printf("[%s] %s",
			blue(name),
			red("FAILED → "+resp.Status),
		)
		return false
	}
}