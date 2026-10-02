// Package chart renders data as text: Sparkline, BarChart, LineChart, HeatMap
// and Gauge. Like widgets, each is a stateless function from its data and a
// theme to a string, with no Msg handling, and the package never imports tui or
// a stateful component. It is split from widgets because charts share one
// concern (turning numbers into marks) and one scaling convention (the slice's
// own min and max), where widgets holds the chrome around content (Badge, Box,
// Divider, ProgressBar, ChatMessage, ...).
//
// Each chart also has a Linearize function (LinearizeSparkline, ...) returning
// a plain-text summary of its data for accessible output.
//
// Stability: stable-ish.
package chart
