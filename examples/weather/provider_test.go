package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestFixtureSearch(t *testing.T) {
	f := fastFixture()
	res, err := f.search(context.Background(), " ber ")
	if err != nil || len(res) != 1 || res[0].Name != "Berlin" {
		t.Fatalf("res = %v err = %v", res, err)
	}
	if res, _ = f.search(context.Background(), "o"); len(res) < 5 {
		t.Fatalf("a short query should match several: %v", res)
	}
	if res, _ = f.search(context.Background(), "zzzz"); len(res) != 0 {
		t.Fatalf("res = %v", res)
	}
}

func TestFixtureForecastIsSevenSensibleDays(t *testing.T) {
	f := fastFixture()
	for _, c := range cities[:10] {
		fc, err := f.forecast(context.Background(), c)
		if c.Name == "Reykjavik" {
			fc, err = f.forecast(context.Background(), c) // it fails the first time
		}
		if err != nil || len(fc.Days) != 7 {
			t.Fatalf("%s: %d days, err %v", c.Name, len(fc.Days), err)
		}
		for i, d := range fc.Days {
			if d.Lo >= d.Hi || d.Rain < 0 || d.Wind < 0 || d.Date.Day() != 14+i {
				t.Fatalf("%s day %d: %+v", c.Name, i, d)
			}
		}
	}
	a, _ := f.forecast(context.Background(), cities[0])
	b, _ := f.forecast(context.Background(), cities[0])
	if a.Days[3] != b.Days[3] {
		t.Fatal("the same place on the same day gave different weather")
	}
	o, _ := f.forecast(context.Background(), cities[2])
	if o.Days[0] == a.Days[0] {
		t.Fatal("two places gave the same weather")
	}
}

func TestFixtureMisbehaviour(t *testing.T) {
	f := fastFixture()
	ctx := context.Background()
	rey := place{Name: "Reykjavik"}
	if _, err := f.forecast(ctx, rey); err == nil || !strings.Contains(err.Error(), "did not answer in time") {
		t.Fatalf("first Reykjavik: %v", err)
	}
	if _, err := f.forecast(ctx, rey); err != nil {
		t.Fatalf("second Reykjavik: %v", err)
	}
	if _, err := f.forecast(ctx, place{Name: "Atlantis"}); !errors.Is(err, errNotFound) {
		t.Fatalf("Atlantis: %v", err)
	}
}

func TestSlowvilleOnlyEndsWhenCancelled(t *testing.T) {
	f := fastFixture()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := f.forecast(ctx, place{Name: "Slowville"}); done <- err }()
	select {
	case err := <-done:
		t.Fatalf("Slowville answered: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancelling did not stop it")
	}
}

func TestDefaultDelaysAreShortAndStable(t *testing.T) {
	f := newFixture()
	for _, c := range cities {
		d := f.delay(c)
		if d < 300*time.Millisecond || d >= 900*time.Millisecond || d != f.delay(c) {
			t.Errorf("%s delay = %v", c.Name, d)
		}
	}
}

func TestSky(t *testing.T) {
	for code, want := range map[int]string{0: "Clear", 1: "Partly cloudy", 3: "Overcast", 45: "Fog", 53: "Drizzle", 63: "Rain", 81: "Rain", 73: "Snow", 86: "Snow", 95: "Thunderstorm", 999: "Thunderstorm", 4: "Unknown"} {
		if _, got := sky(code); got != want {
			t.Errorf("sky(%d) = %q, want %q", code, got, want)
		}
	}
	for c := 0; c <= 99; c++ {
		if g, w := sky(c); g == "" || w == "" {
			t.Errorf("sky(%d) has no symbol or words", c)
		}
	}
}

func TestUnits(t *testing.T) {
	if metric.temp(21.6) != "22°C" || imperial.temp(0) != "32°F" || imperial.temp(100) != "212°F" {
		t.Errorf("temp: %s %s %s", metric.temp(21.6), imperial.temp(0), imperial.temp(100))
	}
	if metric.wind(20) != "20 km/h" || imperial.wind(100) != "62 mph" {
		t.Errorf("wind: %s %s", metric.wind(20), imperial.wind(100))
	}
	if metric.rain(1.25) != "1.2 mm" && metric.rain(1.25) != "1.3 mm" || imperial.rain(25.4) != "1.00 in" {
		t.Errorf("rain: %s %s", metric.rain(1.25), imperial.rain(25.4))
	}
	if (place{Name: "X"}).String() != "X" || (place{Name: "X", Country: "Y"}).String() != "X, Y" {
		t.Error("place string")
	}
}

// ---- Open-Meteo, against a local server ------------------------------------

func meteo(t *testing.T, h http.HandlerFunc) *openMeteo {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	o := newOpenMeteo()
	o.geocodeURL, o.apiURL = srv.URL+"/geo", srv.URL+"/forecast"
	return o
}

const goodForecast = `{"current":{"temperature_2m":12.5,"weather_code":3,"wind_speed_10m":9.5},
"daily":{"time":["2026-10-14","2026-10-15"],"weather_code":[3,61],"temperature_2m_max":[14,11],"temperature_2m_min":[8,7],
"precipitation_sum":[0,4.2],"wind_speed_10m_max":[20,31]}}`

func TestOpenMeteoSearch(t *testing.T) {
	var gotQuery string
	o := meteo(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("name")
		_, _ = w.Write([]byte(`{"results":[{"name":"Berlin","country":"Germany","latitude":52.52,"longitude":13.41},{"name":"Berlin","country":"USA","latitude":44.4,"longitude":-71.2}]}`))
	})
	res, err := o.search(context.Background(), " Ber lin ")
	if err != nil || len(res) != 2 || res[0].Country != "Germany" || res[0].Lat != 52.52 {
		t.Fatalf("res = %v err = %v", res, err)
	}
	if gotQuery != "Ber lin" {
		t.Fatalf("the query reached the server as %q", gotQuery)
	}
	if res, err = o.search(context.Background(), "   "); res != nil || err != nil {
		t.Fatalf("a blank query asked the server: %v %v", res, err)
	}
	empty := meteo(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{}`)) })
	if res, err = empty.search(context.Background(), "x"); err != nil || len(res) != 0 {
		t.Fatalf("no results: %v %v", res, err)
	}
}

func TestOpenMeteoForecast(t *testing.T) {
	var gotPath string
	o := meteo(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.RawQuery
		_, _ = w.Write([]byte(goodForecast))
	})
	fc, err := o.forecast(context.Background(), place{Name: "Berlin", Lat: 52.52, Lon: 13.41})
	if err != nil || len(fc.Days) != 2 {
		t.Fatalf("fc = %+v err = %v", fc, err)
	}
	if fc.NowTemp != 12.5 || fc.NowCode != 3 || fc.Days[1].Code != 61 || fc.Days[1].Rain != 4.2 || fc.Days[0].Date.Day() != 14 || fc.Provider != "open-meteo.com" {
		t.Fatalf("fc = %+v", fc)
	}
	if !strings.Contains(gotPath, "latitude=52.5200") || !strings.Contains(gotPath, "forecast_days=7") {
		t.Fatalf("query = %s", gotPath)
	}
}

func TestOpenMeteoFailures(t *testing.T) {
	for name, h := range map[string]http.HandlerFunc{
		"500":      func(w http.ResponseWriter, r *http.Request) { http.Error(w, "boom", 500) },
		"not json": func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("<html>")) },
		"short": func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"daily":{"time":["2026-10-14"],"weather_code":[]}}`))
		},
		"empty": func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"daily":{}}`)) },
		"bad date": func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(strings.Replace(goodForecast, "2026-10-14", "yesterday", 1)))
		},
	} {
		o := meteo(t, h)
		if _, err := o.forecast(context.Background(), place{Name: "X"}); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
	o := meteo(t, func(w http.ResponseWriter, r *http.Request) { http.Error(w, "x", 503) })
	if _, err := o.search(context.Background(), "a"); err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("search err = %v", err)
	}
}

func TestOpenMeteoStopsWhenCancelled(t *testing.T) {
	block := make(chan struct{})
	o := meteo(t, func(w http.ResponseWriter, r *http.Request) { <-block })
	defer close(block)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	began := time.Now()
	if _, err := o.forecast(ctx, place{Name: "X"}); err == nil || time.Since(began) > 3*time.Second {
		t.Fatalf("err = %v after %v", err, time.Since(began))
	}
}

func TestOpenMeteoLimitsWhatItReads(t *testing.T) {
	o := meteo(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"results":[`))
		for i := 0; i < 200000; i++ {
			_, _ = w.Write([]byte(`{"name":"x","country":"y","latitude":1,"longitude":2},`))
		}
	})
	if res, err := o.search(context.Background(), "x"); err == nil && len(res) > 100000 {
		t.Fatalf("read %d results from an endless answer", len(res))
	}
}

// countingProvider counts the searches that reach the fixture.
type countingProvider struct {
	*fixture
	mu sync.Mutex
	n  int
}

func (c *countingProvider) search(ctx context.Context, q string) ([]place, error) {
	c.mu.Lock()
	c.n++
	c.mu.Unlock()
	return c.fixture.search(ctx, q)
}

func (c *countingProvider) searches() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}
