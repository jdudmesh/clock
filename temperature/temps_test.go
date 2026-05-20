package temperature

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
)

func TestFetch(t *testing.T) {
	mockResponse := thingListResponse{
		Error: 0,
		Msg:   "",
		Data: thingListData{
			ThingList: []thing{
				{
					ItemType: 1,
					ItemData: itemData{
						Name:     "Outside Temperature",
						DeviceID: "a4800a8558",
						Online:   true,
						Params: params{
							Battery:     39,
							Temperature: "1160",
							Humidity:    "8830",
						},
					},
					Index: 0,
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("unexpected Authorization header: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-CK-Appid") != "test-app-id" {
			t.Errorf("unexpected X-CK-Appid header: %s", r.Header.Get("X-CK-Appid"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	logger := zerolog.Nop()
	temp := &Temperature{
		logger:  &logger,
		client:  server.Client(),
		baseURL: server.URL,
		config: &Config{
			AppID:    "test-app-id",
			DeviceID: "test-device-id",
			Token:    "test-token",
		},
	}

	temp.Fetch()

	gotTemp, gotHumi, gotBattery, gotTime := temp.Get()

	if gotTemp != 11.60 {
		t.Errorf("temperature: got %v, want 11.60", gotTemp)
	}
	if gotHumi != 88.30 {
		t.Errorf("humidity: got %v, want 88.30", gotHumi)
	}
	if gotBattery != 39 {
		t.Errorf("battery: got %v, want 39", gotBattery)
	}
	if gotTime.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestFetchAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(thingListResponse{Error: 401, Msg: "unauthorized"})
	}))
	defer server.Close()

	logger := zerolog.Nop()
	temp := &Temperature{
		logger:  &logger,
		client:  server.Client(),
		baseURL: server.URL,
		config: &Config{
			AppID:    "test-app-id",
			DeviceID: "test-device-id",
			Token:    "bad-token",
		},
	}

	temp.Fetch()

	_, _, _, gotTime := temp.Get()
	if !gotTime.IsZero() {
		t.Error("expected zero timestamp on API error")
	}
}

func TestFetchEmptyList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(thingListResponse{
			Error: 0,
			Data:  thingListData{ThingList: []thing{}},
		})
	}))
	defer server.Close()

	logger := zerolog.Nop()
	temp := &Temperature{
		logger:  &logger,
		client:  server.Client(),
		baseURL: server.URL,
		config: &Config{
			AppID:    "test-app-id",
			DeviceID: "test-device-id",
			Token:    "test-token",
		},
	}

	temp.Fetch()

	_, _, _, gotTime := temp.Get()
	if !gotTime.IsZero() {
		t.Error("expected zero timestamp on empty device list")
	}
}
