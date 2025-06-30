package almanac

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

const url = "https://api.sunrisesunset.io/json?lat=48.744760&lng=-0.962368&timezone=UTC&time_format=24"

type AlmanacResults struct {
	Date       string `json:"date"`
	Sunrise    string `json:"sunrise"`
	Sunset     string `json:"sunset"`
	FirstLight string `json:"first_light"`
	LastLight  string `json:"last_light"`
	Dawn       string `json:"dawn"`
	Dusk       string `json:"dusk"`
	SolarNoon  string `json:"solar_noon"`
	GoldenHour string `json:"golden_hour"`
	DayLength  string `json:"day_length"`
	Timezone   string `json:"timezone"`
	UTCOffset  int    `json:"utc_offset"`
}

type AlmanacResponse struct {
	Results AlmanacResults `json:"results"`
	Status  string         `json:"status"`
}

type Almanac struct {
	data   *AlmanacResults
	lock   sync.Mutex
	logger *zerolog.Logger
	client *http.Client
}

func New(logger *zerolog.Logger) *Almanac {
	s := &Almanac{
		lock:   sync.Mutex{},
		logger: logger,
		client: http.DefaultClient,
	}
	return s
}

func (s *Almanac) Fetch() {
	if s.data != nil {
		sunsetDate, err := time.Parse("2006-01-02", s.data.Date)
		if err != nil {
			s.logger.Error().Err(fmt.Errorf("parsing sunset date: %w", err))
			return
		}
		now := time.Now().UTC()
		if now.Year() == sunsetDate.Year() && now.Month() == sunsetDate.Month() && now.Day() == sunsetDate.Day() {
			return
		}
	}

	ctx, cancelFn := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelFn()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		s.logger.Error().Err(fmt.Errorf("creating fetch request: %w", err))
		return
	}

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Error().Err(fmt.Errorf("executing fetch request: %w", err))
		return
	}

	if resp.StatusCode != 200 {
		s.logger.Error().Err(fmt.Errorf("go status: %d", resp.StatusCode))
		return
	}

	data := &AlmanacResponse{}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(data)
	if err != nil {
		s.logger.Error().Err(fmt.Errorf("decoding response: %w", err))
		return
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	s.data = &data.Results
}

func (s *Almanac) GetSunrise() time.Time {
	s.lock.Lock()
	defer s.lock.Unlock()

	sunrise, err := time.ParseInLocation("2006-01-02 15:04:05", s.data.Date+" "+s.data.Sunrise, time.UTC)
	if err != nil {
		s.logger.Error().Err(fmt.Errorf("parsing sunrise: %w", err))
		return time.Time{}
	}

	return sunrise
}

func (s *Almanac) GetSunset() time.Time {
	s.lock.Lock()
	defer s.lock.Unlock()

	sunset, err := time.ParseInLocation("2006-01-02 15:04:05", s.data.Date+" "+s.data.Sunset, time.UTC)
	if err != nil {
		s.logger.Error().Err(fmt.Errorf("parsing sunset: %w", err))
		return time.Time{}
	}

	return sunset
}

func (s *Almanac) Run(ctx context.Context) {
	s.Fetch()
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(1 * time.Minute):
			s.Fetch()
		}
	}
}
