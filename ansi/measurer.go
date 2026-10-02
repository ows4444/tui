package ansi

// Measurer measures text the way one particular terminal draws it. The
// package-level Width, Truncate and TrimLeftWidth use a single process-wide
// setting (SetClusterWidth); a Measurer carries its own, so two terminals in
// one process (an SSH server, parallel tests) can disagree about grapheme
// clusters without one changing the other's output.
//
// The zero Measurer follows the process setting, so a Measurer that was never
// configured measures exactly like the package-level functions. ClusterMeasurer
// pins it either way. A Measurer is a small comparable value; copy it freely.
type Measurer struct {
	mode measureMode
}

type measureMode uint8

const (
	measureDefault measureMode = iota // follow SetClusterWidth / TUI_NO_CLUSTERS
	measureClusters
	measureRunes
)

// ClusterMeasurer returns a Measurer that treats an extended grapheme cluster
// as one unit when clusters is true and counts each rune on its own when it is
// false, whatever SetClusterWidth says. See SetClusterWidth for what a cluster
// measures.
func ClusterMeasurer(clusters bool) Measurer {
	if clusters {
		return Measurer{measureClusters}
	}
	return Measurer{measureRunes}
}

// Clusters reports whether m treats a grapheme cluster as one unit.
func (m Measurer) Clusters() bool {
	switch m.mode {
	case measureClusters:
		return true
	case measureRunes:
		return false
	}
	return clusterWidthOn.Load()
}

// Width is Width measured by m.
func (m Measurer) Width(s string) int { return width(s, m.Clusters()) }

// Truncate is Truncate measured by m.
func (m Measurer) Truncate(s string, w int) string { return truncate(s, w, m.Clusters()) }

// TrimLeftWidth is TrimLeftWidth measured by m.
func (m Measurer) TrimLeftWidth(s string, w int) string { return trimLeftWidth(s, w, m.Clusters()) }
