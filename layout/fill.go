package layout

// Fill returns a FlexChild that takes a share of whatever main-axis space is
// left after its siblings are sized, however large or small the node itself
// measures. It is Basis: 1 with Grow: 1 (and Shrink: 1, so it also gives
// space back when the container is too small), which is the idiom scrollable
// content needs: a viewport or a long table measures to its full length, and
// a growing child that measures larger than the screen would otherwise push
// its siblings out of view. Several Fill children split the leftover equally;
// use FillWeight for unequal shares.
func Fill(n Node) FlexChild { return FillWeight(n, 1) }

// FillWeight is Fill with a share weight: among Fill children, leftover space
// is split in proportion to weight. A weight below 1 counts as 1.
func FillWeight(n Node, weight int) FlexChild {
	if weight < 1 {
		weight = 1
	}
	return FlexChild{Node: n, Basis: 1, Grow: weight, Shrink: 1}
}
