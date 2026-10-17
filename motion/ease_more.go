package motion

import (
	"math"
	"sort"
	"strings"
)

// The rest of the standard easing set (the families of easings.net). Like the
// first eight in ease.go, each returns 0 at 0 and 1 at 1 and clamps t. Back and
// Elastic overshoot, so they are not monotonic.

func inPow(n float64) Ease  { return func(t float64) float64 { return math.Pow(clamp01(t), n) } }
func outPow(n float64) Ease { return func(t float64) float64 { return 1 - math.Pow(1-clamp01(t), n) } }
func inOutPow(n float64) Ease {
	return func(t float64) float64 {
		t = clamp01(t)
		if t < 0.5 {
			return math.Pow(2, n-1) * math.Pow(t, n)
		}
		return 1 - math.Pow(-2*t+2, n)/2
	}
}

// InQuart, OutQuart and InOutQuart are the fourth-power curves.
var (
	InQuart    Ease = inPow(4)
	OutQuart   Ease = outPow(4)
	InOutQuart Ease = inOutPow(4)
	// InQuint, OutQuint and InOutQuint are the fifth-power curves.
	InQuint    Ease = inPow(5)
	OutQuint   Ease = outPow(5)
	InOutQuint Ease = inOutPow(5)
)

// InSine eases in along a quarter sine wave.
func InSine(t float64) float64 { return 1 - math.Cos(clamp01(t)*math.Pi/2) }

// OutSine eases out along a quarter sine wave.
func OutSine(t float64) float64 { return math.Sin(clamp01(t) * math.Pi / 2) }

// InOutSine eases in and out along half a cosine wave.
func InOutSine(t float64) float64 { return -(math.Cos(math.Pi*clamp01(t)) - 1) / 2 }

// InExpo starts very slowly and ends steeply (2 to the power of 10t).
func InExpo(t float64) float64 {
	if t = clamp01(t); t == 0 {
		return 0
	}
	return math.Pow(2, 10*t-10)
}

// OutExpo starts steeply and settles very slowly.
func OutExpo(t float64) float64 {
	if t = clamp01(t); t == 1 {
		return 1
	}
	return 1 - math.Pow(2, -10*t)
}

// InOutExpo is InExpo then OutExpo.
func InOutExpo(t float64) float64 {
	switch t = clamp01(t); {
	case t == 0:
		return 0
	case t == 1:
		return 1
	case t < 0.5:
		return math.Pow(2, 20*t-10) / 2
	}
	return (2 - math.Pow(2, -20*t+10)) / 2
}

// InCirc eases in along a circle's arc.
func InCirc(t float64) float64 { t = clamp01(t); return 1 - math.Sqrt(1-t*t) }

// OutCirc eases out along a circle's arc.
func OutCirc(t float64) float64 { u := clamp01(t) - 1; return math.Sqrt(1 - u*u) }

// InOutCirc is InCirc then OutCirc.
func InOutCirc(t float64) float64 {
	t = clamp01(t)
	if t < 0.5 {
		return (1 - math.Sqrt(1-math.Pow(2*t, 2))) / 2
	}
	return (math.Sqrt(1-math.Pow(-2*t+2, 2)) + 1) / 2
}

const (
	backC1 = 1.70158
	backC2 = backC1 * 1.525
	backC3 = backC1 + 1
)

// InBack pulls back below 0 before moving forward.
func InBack(t float64) float64 { t = clamp01(t); return backC3*t*t*t - backC1*t*t }

// InOutBack pulls back, moves forward, overshoots 1 and settles.
func InOutBack(t float64) float64 {
	t = clamp01(t)
	if t < 0.5 {
		return (math.Pow(2*t, 2) * ((backC2+1)*2*t - backC2)) / 2
	}
	return (math.Pow(2*t-2, 2)*((backC2+1)*(t*2-2)+backC2) + 2) / 2
}

// InElastic swings like a spring pulled back and released, growing in.
func InElastic(t float64) float64 {
	switch t = clamp01(t); {
	case t == 0:
		return 0
	case t == 1:
		return 1
	}
	return -math.Pow(2, 10*t-10) * math.Sin((t*10-10.75)*(2*math.Pi/3))
}

// OutElastic overshoots and rings down onto 1.
func OutElastic(t float64) float64 {
	switch t = clamp01(t); {
	case t == 0:
		return 0
	case t == 1:
		return 1
	}
	return math.Pow(2, -10*t)*math.Sin((t*10-0.75)*(2*math.Pi/3)) + 1
}

// InOutElastic is InElastic then OutElastic.
func InOutElastic(t float64) float64 {
	const c5 = 2 * math.Pi / 4.5
	switch t = clamp01(t); {
	case t == 0:
		return 0
	case t == 1:
		return 1
	case t < 0.5:
		return -(math.Pow(2, 20*t-10) * math.Sin((20*t-11.125)*c5)) / 2
	}
	return (math.Pow(2, -20*t+10)*math.Sin((20*t-11.125)*c5))/2 + 1
}

// OutBounce drops onto 1 and bounces to rest.
func OutBounce(t float64) float64 {
	const n1, d1 = 7.5625, 2.75
	t = clamp01(t)
	switch {
	case t < 1/d1:
		return n1 * t * t
	case t < 2/d1:
		t -= 1.5 / d1
		return n1*t*t + 0.75
	case t < 2.5/d1:
		t -= 2.25 / d1
		return n1*t*t + 0.9375
	}
	t -= 2.625 / d1
	return n1*t*t + 0.984375
}

// InBounce is OutBounce played backwards.
func InBounce(t float64) float64 { return 1 - OutBounce(1-clamp01(t)) }

// InOutBounce bounces at both ends.
func InOutBounce(t float64) float64 {
	if t = clamp01(t); t < 0.5 {
		return (1 - OutBounce(1-2*t)) / 2
	}
	return (1 + OutBounce(2*t-1)) / 2
}

var easeByName = map[string]Ease{
	"linear":       Linear,
	"in-quad":      InQuad,
	"out-quad":     OutQuad,
	"in-out-quad":  InOutQuad,
	"in-cubic":     InCubic,
	"out-cubic":    OutCubic,
	"in-out-cubic": InOutCubic,
	"in-quart":     InQuart, "out-quart": OutQuart, "in-out-quart": InOutQuart,
	"in-quint": InQuint, "out-quint": OutQuint, "in-out-quint": InOutQuint,
	"in-sine": InSine, "out-sine": OutSine, "in-out-sine": InOutSine,
	"in-expo": InExpo, "out-expo": OutExpo, "in-out-expo": InOutExpo,
	"in-circ": InCirc, "out-circ": OutCirc, "in-out-circ": InOutCirc,
	"in-back": InBack, "out-back": OutBack, "in-out-back": InOutBack,
	"in-elastic": InElastic, "out-elastic": OutElastic, "in-out-elastic": InOutElastic,
	"in-bounce": InBounce, "out-bounce": OutBounce, "in-out-bounce": InOutBounce,
}

// ByName returns the easing with the given name, such as "out-cubic" or
// "in-out-elastic" (see EaseNames; case and "_" for "-" do not matter), so a
// config file or a theme can pick one. ok is false for an unknown name.
func ByName(name string) (Ease, bool) {
	e, ok := easeByName[strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), "_", "-")]
	return e, ok
}

// EaseNames lists the names ByName accepts, sorted.
func EaseNames() []string {
	names := make([]string, 0, len(easeByName))
	for n := range easeByName {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
