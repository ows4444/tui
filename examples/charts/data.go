package main

import (
	"fmt"
	"math"
	"math/rand"
)

// dataset is a named group of series that share an x axis.
type dataset struct {
	name   string
	unit   string
	x      func(i int) string // the x axis label of point i
	series []seriesData
}

type seriesData struct {
	name   string
	values []float64
}

func months(i int) string {
	return [...]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}[i%12]
}

func index(i int) string { return fmt.Sprint(i) }

func seconds(i int) string { return fmt.Sprintf("%ds", i) }

// datasets are the made-up numbers the explorer opens with.
func datasets() []dataset {
	rng := rand.New(rand.NewSource(7)) // #nosec G404 -- made-up numbers, not security-sensitive
	lat50, lat99 := make([]float64, 300), make([]float64, 300)
	for i := range lat50 {
		base := 40 + 8*math.Sin(float64(i)/20)
		lat50[i] = base * math.Exp(rng.NormFloat64()*0.15)
		lat99[i] = lat50[i] * (2.5 + rng.Float64()*1.5)
		if i%97 == 60 {
			lat99[i] *= 3 // a spike
		}
	}
	waves := make([][]float64, 3)
	for k := range waves {
		waves[k] = make([]float64, 200)
		for i := range waves[k] {
			x := float64(i) / 200 * 4 * math.Pi
			switch k {
			case 0:
				waves[k][i] = math.Sin(x)
			case 1:
				waves[k][i] = math.Cos(x) * 0.7
			default:
				waves[k][i] = math.Sin(x)*math.Cos(x*3)*0.5 + 0.2
			}
		}
	}
	growth := [3][]float64{make([]float64, 60), make([]float64, 60), make([]float64, 60)}
	for i := range growth[0] {
		growth[0][i] = 10 * math.Pow(1.08, float64(i))
		growth[1][i] = 10 + 12*float64(i)
		growth[2][i] = 1000 / (1 + math.Exp(-(float64(i)-30)/6))
	}
	return []dataset{
		{name: "Temperature", unit: "°C", x: months, series: []seriesData{
			{"Berlin", []float64{0, 1, 5, 9, 14, 17, 19, 19, 15, 10, 5, 1}},
			{"Cairo", []float64{14, 15, 18, 22, 26, 28, 29, 29, 27, 24, 19, 15}},
			{"Oslo", []float64{-4, -4, 0, 5, 11, 15, 17, 16, 11, 6, 1, -2}},
		}},
		{name: "Latency", unit: "ms", x: seconds, series: []seriesData{{"p50", lat50}, {"p99", lat99}}},
		{name: "Waves", unit: "", x: index, series: []seriesData{{"sin", waves[0]}, {"cos", waves[1]}, {"mix", waves[2]}}},
		{name: "Growth", unit: "", x: index, series: []seriesData{{"compound", growth[0]}, {"linear", growth[1]}, {"logistic", growth[2]}}},
	}
}

func (d dataset) points() int { return len(d.series[0].values) }

func (d dataset) values() [][]float64 {
	out := make([][]float64, len(d.series))
	for i, s := range d.series {
		out[i] = s.values
	}
	return out
}
