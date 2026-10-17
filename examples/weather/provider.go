package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// place is somewhere a forecast can be had for.
type place struct {
	Name, Country string
	Lat, Lon      float64
}

func (p place) String() string {
	if p.Country == "" {
		return p.Name
	}
	return p.Name + ", " + p.Country
}

// day is one day of a forecast, in metric units.
type day struct {
	Date time.Time
	Code int     // WMO weather code
	Hi   float64 // °C
	Lo   float64
	Rain float64 // mm
	Wind float64 // km/h
}

type forecast struct {
	Place    place
	NowTemp  float64
	NowWind  float64
	NowCode  int
	Days     []day
	Fetched  time.Time
	Provider string
}

// provider is where places and forecasts come from.
type provider interface {
	name() string
	search(ctx context.Context, q string) ([]place, error)
	forecast(ctx context.Context, p place) (forecast, error)
}

var errNotFound = errors.New("no forecast for that place (404)")

// ---- the fixture -----------------------------------------------------------

// fixture answers from a small built-in table after a short wait, so the
// example works offline and the waiting can be seen. Three places misbehave on
// purpose: Reykjavik times out the first time it is asked, Atlantis does not
// exist, and Slowville takes so long that cancelling it is the only way out.
type fixture struct {
	delay func(p place) time.Duration
	now   func() time.Time
	mu    sync.Mutex
	tried map[string]int
}

func newFixture() *fixture {
	return &fixture{
		delay: func(p place) time.Duration {
			h := fnv.New32a()
			h.Write([]byte(p.Name))
			return 300*time.Millisecond + time.Duration(h.Sum32()%600)*time.Millisecond
		},
		now:   time.Now,
		tried: map[string]int{},
	}
}

var cities = []place{
	{"Berlin", "Germany", 52.52, 13.41}, {"Cairo", "Egypt", 30.04, 31.24}, {"Oslo", "Norway", 59.91, 10.75},
	{"Tokyo", "Japan", 35.68, 139.69}, {"Lima", "Peru", -12.05, -77.04}, {"Nairobi", "Kenya", -1.29, 36.82},
	{"Reykjavik", "Iceland", 64.15, -21.94}, {"Sydney", "Australia", -33.87, 151.21}, {"Toronto", "Canada", 43.65, -79.38},
	{"Mumbai", "India", 19.08, 72.88}, {"Atlantis", "Nowhere", 0, 0}, {"Slowville", "Patience", 1, 1},
}

func (f *fixture) name() string { return "built-in fixture" }

func (f *fixture) wait(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (f *fixture) search(ctx context.Context, q string) ([]place, error) {
	if err := f.wait(ctx, 80*time.Millisecond); err != nil {
		return nil, err
	}
	q = strings.ToLower(strings.TrimSpace(q))
	var out []place
	for _, c := range cities {
		if q != "" && strings.Contains(strings.ToLower(c.Name), q) {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fixture) forecast(ctx context.Context, p place) (forecast, error) {
	f.mu.Lock()
	f.tried[p.Name]++
	attempt := f.tried[p.Name]
	f.mu.Unlock()
	d := f.delay(p)
	if p.Name == "Slowville" {
		d = time.Minute
	}
	if err := f.wait(ctx, d); err != nil {
		return forecast{}, err
	}
	switch {
	case p.Name == "Atlantis":
		return forecast{}, errNotFound
	case p.Name == "Reykjavik" && attempt == 1:
		return forecast{}, errors.New("the weather service did not answer in time")
	}
	return f.build(p), nil
}

// build makes seven days of weather that depend on the place and the date.
func (f *fixture) build(p place) forecast {
	now := f.now()
	h := fnv.New32a()
	h.Write([]byte(p.Name))
	seed := float64(h.Sum32() % 1000)
	base := 28 - math.Abs(p.Lat)*0.5 // colder away from the equator
	fc := forecast{Place: p, Fetched: now, Provider: f.name()}
	for i := 0; i < 7; i++ {
		x := seed + float64(i)*1.7
		hi := base + 4*math.Sin(x)
		code := []int{0, 1, 2, 3, 3, 61, 63, 71, 95, 45}[int(math.Abs(math.Sin(x*2.3))*10)%10]
		rain := 0.0
		if code >= 51 {
			rain = math.Round(math.Abs(math.Cos(x))*80) / 10
		}
		if code == 71 {
			hi = math.Min(hi, 0.5)
		}
		fc.Days = append(fc.Days, day{
			Date: time.Date(now.Year(), now.Month(), now.Day()+i, 0, 0, 0, 0, time.UTC),
			Code: code, Hi: math.Round(hi*10) / 10, Lo: math.Round((hi-6-2*math.Abs(math.Cos(x)))*10) / 10,
			Rain: rain, Wind: math.Round(8 + 20*math.Abs(math.Sin(x*0.7))),
		})
	}
	fc.NowTemp, fc.NowCode, fc.NowWind = fc.Days[0].Hi-2, fc.Days[0].Code, fc.Days[0].Wind
	return fc
}

// ---- Open-Meteo ------------------------------------------------------------

// openMeteo reads the free Open-Meteo services. It is used only with -live.
type openMeteo struct {
	client             *http.Client
	geocodeURL, apiURL string
}

func newOpenMeteo() *openMeteo {
	return &openMeteo{
		client:     &http.Client{Timeout: 10 * time.Second},
		geocodeURL: "https://geocoding-api.open-meteo.com/v1/search",
		apiURL:     "https://api.open-meteo.com/v1/forecast",
	}
}

func (o *openMeteo) name() string { return "open-meteo.com" }

const maxBody = 1 << 20

func (o *openMeteo) get(ctx context.Context, u string, into any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := o.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("the service answered %s", resp.Status)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBody)).Decode(into); err != nil {
		return fmt.Errorf("the service sent something unreadable: %w", err)
	}
	return nil
}

func (o *openMeteo) search(ctx context.Context, q string) ([]place, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, nil
	}
	var r struct {
		Results []struct {
			Name    string  `json:"name"`
			Country string  `json:"country"`
			Lat     float64 `json:"latitude"`
			Lon     float64 `json:"longitude"`
		} `json:"results"`
	}
	if err := o.get(ctx, o.geocodeURL+"?count=6&name="+url.QueryEscape(q), &r); err != nil {
		return nil, err
	}
	var out []place
	for _, x := range r.Results {
		out = append(out, place{x.Name, x.Country, x.Lat, x.Lon})
	}
	return out, nil
}

func (o *openMeteo) forecast(ctx context.Context, p place) (forecast, error) {
	var r struct {
		Current struct {
			Temp float64 `json:"temperature_2m"`
			Code int     `json:"weather_code"`
			Wind float64 `json:"wind_speed_10m"`
		} `json:"current"`
		Daily struct {
			Time []string  `json:"time"`
			Code []int     `json:"weather_code"`
			Hi   []float64 `json:"temperature_2m_max"`
			Lo   []float64 `json:"temperature_2m_min"`
			Rain []float64 `json:"precipitation_sum"`
			Wind []float64 `json:"wind_speed_10m_max"`
		} `json:"daily"`
	}
	u := fmt.Sprintf("%s?latitude=%.4f&longitude=%.4f&timezone=auto&forecast_days=7"+
		"&current=temperature_2m,weather_code,wind_speed_10m"+
		"&daily=weather_code,temperature_2m_max,temperature_2m_min,precipitation_sum,wind_speed_10m_max", o.apiURL, p.Lat, p.Lon)
	if err := o.get(ctx, u, &r); err != nil {
		return forecast{}, err
	}
	d := r.Daily
	n := len(d.Time)
	if n == 0 || len(d.Code) < n || len(d.Hi) < n || len(d.Lo) < n || len(d.Rain) < n || len(d.Wind) < n {
		return forecast{}, errors.New("the service sent an incomplete forecast")
	}
	fc := forecast{Place: p, NowTemp: r.Current.Temp, NowCode: r.Current.Code, NowWind: r.Current.Wind, Fetched: time.Now(), Provider: o.name()}
	for i := 0; i < n; i++ {
		t, err := time.Parse("2006-01-02", d.Time[i])
		if err != nil {
			return forecast{}, errors.New("the service sent a date that cannot be read")
		}
		fc.Days = append(fc.Days, day{Date: t, Code: d.Code[i], Hi: d.Hi[i], Lo: d.Lo[i], Rain: d.Rain[i], Wind: d.Wind[i]})
	}
	return fc, nil
}

// ---- weather codes and units -----------------------------------------------

// sky is a WMO weather code as a symbol and words. The words matter: the symbol
// alone is not enough in a font without it, or for a screen reader.
func sky(code int) (string, string) {
	switch {
	case code == 0:
		return "☀", "Clear"
	case code <= 2:
		return "◐", "Partly cloudy"
	case code == 3:
		return "☁", "Overcast"
	case code == 45 || code == 48:
		return "≋", "Fog"
	case code >= 51 && code <= 57:
		return "☂", "Drizzle"
	case code >= 61 && code <= 67, code >= 80 && code <= 82:
		return "☂", "Rain"
	case code >= 71 && code <= 77, code == 85 || code == 86:
		return "❄", "Snow"
	case code >= 95:
		return "⚡", "Thunderstorm"
	}
	return "?", "Unknown"
}

type units int

const (
	metric units = iota
	imperial
)

func (u units) temp(c float64) string {
	if u == imperial {
		return fmt.Sprintf("%.0f°F", c*9/5+32)
	}
	return fmt.Sprintf("%.0f°C", c)
}

func (u units) wind(kmh float64) string {
	if u == imperial {
		return fmt.Sprintf("%.0f mph", kmh*0.621371)
	}
	return fmt.Sprintf("%.0f km/h", kmh)
}

func (u units) rain(mm float64) string {
	if u == imperial {
		return fmt.Sprintf("%.2f in", mm/25.4)
	}
	return fmt.Sprintf("%.1f mm", mm)
}
