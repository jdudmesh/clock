package temperature

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"
)

type Temperature struct {
	temperature float64
	humidity    float64
	battery     int
	timestamp   time.Time
	lock        sync.Mutex
	logger      *zerolog.Logger
	client      *http.Client
	config      *Config
	baseURL     string
}

type Config struct {
	AppID    string `envconfig:"EWELINK_APP_ID" required:"true"`
	DeviceID string `envconfig:"EWELINK_DEVICE_ID" required:"true"`
	Token    string `envconfig:"EWELINK_TOKEN" required:"true"`
}

const apiURL = "https://eu-apia.coolkit.cc/v2/device/thing"

type thingListRequest struct {
	ThingList []thingID `json:"thingList"`
}

type thingID struct {
	ID string `json:"id"`
}

type thingListResponse struct {
	Error int           `json:"error"`
	Msg   string        `json:"msg"`
	Data  thingListData `json:"data"`
}

type thingListData struct {
	ThingList []thing `json:"thingList"`
}

type thing struct {
	ItemType int      `json:"itemType"`
	ItemData itemData `json:"itemData"`
	Index    int      `json:"index"`
}

type itemData struct {
	Name         string `json:"name"`
	DeviceID     string `json:"deviceid"`
	APIKey       string `json:"apikey"`
	Extra        extra  `json:"extra"`
	BrandName    string `json:"brandName"`
	ProductModel string `json:"productModel"`
	Online       bool   `json:"online"`
	Params       params `json:"params"`
}

type extra struct {
	Mac           string `json:"mac"`
	ApMac         string `json:"apmac"`
	Model         string `json:"model"`
	Description   string `json:"description"`
	ModelInfo     string `json:"modelInfo"`
	Manufacturer  string `json:"manufacturer"`
	BrandID       string `json:"brandId"`
	UIID          int    `json:"uiid"`
	UI            string `json:"ui"`
	ReportProduct string `json:"reportProduct"`
}

type params struct {
	BindInfos         map[string]interface{} `json:"bindInfos"`
	SubDevID          string                 `json:"subDevId"`
	ParentID          string                 `json:"parentid"`
	FWVersion         string                 `json:"fwVersion"`
	Battery           int                    `json:"battery"`
	TrigTime          string                 `json:"trigTime"`
	SupportPowConfig  int                    `json:"supportPowConfig"`
	SubDevRssi        int                    `json:"subDevRssi"`
	TempUnit          int                    `json:"tempUnit"`
	TempComfortLower  string                 `json:"tempComfortLower"`
	TempComfortUpper  string                 `json:"tempComfortUpper"`
	HumiComfortLower  string                 `json:"humiComfortLower"`
	HumiComfortUpper  string                 `json:"humiComfortUpper"`
	Temperature       string                 `json:"temperature"`
	TempComfortStatus int                    `json:"tempComfortStatus"`
	Humidity          string                 `json:"humidity"`
	HumiComfortStatus int                    `json:"humiComfortStatus"`
	TimeZone          int                    `json:"timeZone"`
	SubDevRssiSetting subDevRssiSetting      `json:"subDevRssiSetting"`
}

type subDevRssiSetting struct {
	Active   int `json:"active"`
	Duration int `json:"duration"`
}

func New(logger *zerolog.Logger) *Temperature {
	var config Config
	err := envconfig.Process("", &config)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to process ewelink environment variables")
		return nil
	}

	return &Temperature{
		lock:    sync.Mutex{},
		logger:  logger,
		client:  http.DefaultClient,
		config:  &config,
		baseURL: apiURL,
	}
}

func (t *Temperature) Fetch() {
	nonce := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	ctx, cancelFn := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelFn()

	body, err := json.Marshal(thingListRequest{
		ThingList: []thingID{{ID: t.config.DeviceID}},
	})
	if err != nil {
		t.logger.Error().Err(fmt.Errorf("encoding request body: %w", err)).Msg("")
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL, bytes.NewReader(body))
	if err != nil {
		t.logger.Error().Err(fmt.Errorf("creating fetch request: %w", err)).Msg("")
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Authorization", "Bearer "+t.config.Token)
	req.Header.Set("X-CK-Nonce", nonce)
	req.Header.Set("X-CK-Appid", t.config.AppID)

	resp, err := t.client.Do(req)
	if err != nil {
		t.logger.Error().Err(fmt.Errorf("executing fetch request: %w", err)).Msg("")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.logger.Error().Msgf("unexpected status: %d", resp.StatusCode)
		return
	}

	var response thingListResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		t.logger.Error().Err(fmt.Errorf("decoding response: %w", err)).Msg("")
		return
	}

	if response.Error != 0 {
		t.logger.Error().Msgf("api error: %s", response.Msg)
		return
	}

	if len(response.Data.ThingList) == 0 {
		t.logger.Error().Msg("empty device list in response")
		return
	}

	p := response.Data.ThingList[0].ItemData.Params

	temperature, err := strconv.ParseFloat(p.Temperature, 64)
	if err != nil {
		t.logger.Error().Err(fmt.Errorf("parsing temperature: %w", err)).Msg("")
		return
	}

	humidity, err := strconv.ParseFloat(p.Humidity, 64)
	if err != nil {
		t.logger.Error().Err(fmt.Errorf("parsing humidity: %w", err)).Msg("")
		return
	}

	t.lock.Lock()
	defer t.lock.Unlock()
	t.temperature = temperature / 100.0
	t.humidity = humidity / 100.0
	t.battery = p.Battery
	t.timestamp = time.Now()
}

func (t *Temperature) Run(ctx context.Context) {
	t.Fetch()
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Minute):
			t.Fetch()
		}
	}
}

func (t *Temperature) Get() (float64, float64, int, time.Time) {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.temperature, t.humidity, t.battery, t.timestamp
}
