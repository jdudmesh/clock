package router

import (
	"fmt"
	"net/http"
	"time"

	"clock/sunset"
	"clock/temperature"

	"github.com/rs/zerolog"
)

func NewRouter(logger *zerolog.Logger, temperature *temperature.Temperature, sunset *sunset.Sunset) *http.ServeMux {
	loc, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		logger.Error().Err(err).Msg("failed to load location")
		loc = time.UTC
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/clock", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/clock.html")
	})

	mux.HandleFunc("/dist.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/dist.css")
	})

	mux.HandleFunc("/snippets/time", func(w http.ResponseWriter, r *http.Request) {
		now := time.Now().In(loc)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<span>" + now.Format("15:04") + "</span>"))
	})

	mux.HandleFunc("/snippets/temperature", func(w http.ResponseWriter, r *http.Request) {
		now := time.Now().UTC()
		temperatureVal, _, battery, timestamp := temperature.Get()
		classes := ""
		if battery < 20 {
			classes = "battery"
		}
		if timestamp.Before(now.Add(-10 * time.Minute)) {
			classes = "error"
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(fmt.Sprintf("<span class='%s'>%0.1f°</span>", classes, temperatureVal)))
	})

	mux.HandleFunc("/snippets/humidity", func(w http.ResponseWriter, r *http.Request) {
		now := time.Now().UTC()
		_, humidity, battery, timestamp := temperature.Get()
		classes := ""
		if battery < 20 {
			classes = "battery"
		}
		if timestamp.Before(now.Add(-10 * time.Minute)) {
			classes = "error"
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(fmt.Sprintf("<span class='%s'>%0.0f%%</span>", classes, humidity)))
	})

	mux.HandleFunc("/snippets/sunrise", func(w http.ResponseWriter, r *http.Request) {
		sunrise, _ := sunset.Get()
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(fmt.Sprintf("<span>↑%s</span>", sunrise.Format("15:04"))))
	})

	mux.HandleFunc("/snippets/sunset", func(w http.ResponseWriter, r *http.Request) {
		_, sunsetVal := sunset.Get()
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(fmt.Sprintf("<span>↓%s</span>", sunsetVal.Format("15:04"))))
	})

	mux.HandleFunc("/snippets/date", func(w http.ResponseWriter, r *http.Request) {
		now := time.Now().In(loc)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(fmt.Sprintf("<span>%s</span>", now.Format("Monday January 2 2006"))))
	})

	mux.HandleFunc("/data.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
	})

	return mux
}
