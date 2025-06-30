package temperature

import (
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
}

type Config struct {
	AppID    string `envconfig:"EWELINK_APP_ID" required:"true"`
	DeviceID string `envconfig:"EWELINK_DEVICE_ID" required:"true"`
	Token    string `envconfig:"EWELINK_TOKEN" required:"true"`
}

const urlFormat = "https://eu-api.coolkit.cc:8080/api/user/device/%s?appid=%s&version=8&deviceid=%s&nonce=%s&ts=%d"

//"https://eu-api.coolkit.cc:8080/api/user/device/a4800a8558?deviceid=a4800a8558&appid=YzfeftUVcZ6twZw1OoVKPRFYTrGEg01Q&nonce=adu5n393&ts=1751269167&version=8"

type Device struct {
	Code         int           `json:"code"`
	Error        string        `json:"error"`
	ID           string        `json:"_id"`
	Name         string        `json:"name"`
	Type         string        `json:"type"`
	APIKey       string        `json:"apikey"`
	DeviceID     string        `json:"deviceid"`
	CreatedAt    string        `json:"createdAt"`
	ShareUsers   []interface{} `json:"shareUsers"`
	Online       bool          `json:"online"`
	Address      string        `json:"address"`
	Extra        Extra         `json:"extra"`
	Params       Params        `json:"params"`
	IP           string        `json:"ip"`
	Location     string        `json:"location"`
	Family       Family        `json:"family"`
	OfflineTime  string        `json:"offlineTime"`
	OnlineTime   string        `json:"onlineTime"`
	BrandName    string        `json:"brandName"`
	ProductModel string        `json:"productModel"`
	UIID         int           `json:"uiid"`
}

type Extra struct {
	ID            string        `json:"_id"`
	Name          string        `json:"name"`
	APIKey        string        `json:"apikey"`
	DeviceID      string        `json:"deviceid"`
	CreatedAt     string        `json:"createdAt"`
	ExtraInfo     ExtraInfo     `json:"extra"`
	PartnerDevice PartnerDevice `json:"partnerDevice"`
}

type ExtraInfo struct {
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

type PartnerDevice struct {
	EzVedioSerial string `json:"ezVedioSerial"`
}

type Params struct {
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
	SubDevRssiSetting SubDevRssiSetting      `json:"subDevRssiSetting"`
}

type SubDevRssiSetting struct {
	Active   int `json:"active"`
	Duration int `json:"duration"`
}

type Family struct {
	ID      string        `json:"id"`
	Index   int           `json:"index"`
	Members []interface{} `json:"members"`
}

func New(logger *zerolog.Logger) *Temperature {
	var config Config
	err := envconfig.Process("", &config)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to process ewelink environment variables")
		return nil
	}

	return &Temperature{
		lock:   sync.Mutex{},
		logger: logger,
		client: http.DefaultClient,
		config: &config,
	}
}

func (t *Temperature) Fetch() {
	nonce := strings.ReplaceAll(uuid.New().String(), "-", "")[:5]

	ctx, cancelFn := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelFn()

	url := fmt.Sprintf(urlFormat, t.config.DeviceID, t.config.AppID, t.config.DeviceID, nonce, time.Now().Unix())
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		t.logger.Error().Err(fmt.Errorf("creating fetch request: %w", err))
		return
	}

	req.Header.Set("Authorization", "Bearer "+t.config.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		t.logger.Error().Err(fmt.Errorf("executing fetch request: %w", err))
		return
	}

	if resp.StatusCode != 200 {
		t.logger.Error().Err(fmt.Errorf("go status: %d", resp.StatusCode))
		return
	}

	defer resp.Body.Close()

	response := &Device{}
	err = json.NewDecoder(resp.Body).Decode(response)
	if err != nil {
		t.logger.Error().Err(fmt.Errorf("decoding response: %w", err))
		return
	}

	if response.Code != 0 {
		t.logger.Error().Msgf("error: %s", response.Error)
		return
	}

	temperature, err := strconv.ParseFloat(response.Params.Temperature, 64)
	if err != nil {
		t.logger.Error().Err(fmt.Errorf("parsing temperature: %w", err))
		return
	}

	humidity, err := strconv.ParseFloat(response.Params.Humidity, 64)
	if err != nil {
		t.logger.Error().Err(fmt.Errorf("parsing humidity: %w", err))
		return
	}

	t.lock.Lock()
	defer t.lock.Unlock()
	t.temperature = temperature / 100.0
	t.humidity = humidity / 100.0
	t.battery = response.Params.Battery
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
