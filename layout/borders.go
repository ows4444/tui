package layout

// More border presets. The source holds only ASCII: the glyphs are \u escapes.

var (
	blockBorder = Border{Top: "█", Bottom: "█", Left: "█", Right: "█",
		TopLeft: "█", TopRight: "█", BottomLeft: "█", BottomRight: "█"}
	outerHalfBlockBorder = Border{Top: "▀", Bottom: "▄", Left: "▌", Right: "▐",
		TopLeft: "▛", TopRight: "▜", BottomLeft: "▙", BottomRight: "▟"}
	innerHalfBlockBorder = Border{Top: "▄", Bottom: "▀", Left: "▐", Right: "▌",
		TopLeft: "▗", TopRight: "▖", BottomLeft: "▝", BottomRight: "▘"}
	dashedBorder = Border{Top: "╌", Bottom: "╌", Left: "╎", Right: "╎",
		TopLeft: "┌", TopRight: "┐", BottomLeft: "└", BottomRight: "┘"}
	heavyDashedBorder = Border{Top: "╍", Bottom: "╍", Left: "╏", Right: "╏",
		TopLeft: "┏", TopRight: "┓", BottomLeft: "┗", BottomRight: "┛"}
	markdownBorder = Border{Top: "-", Bottom: "-", Left: "|", Right: "|",
		TopLeft: "|", TopRight: "|", BottomLeft: "|", BottomRight: "|"}
	hiddenBorder = Border{Top: " ", Bottom: " ", Left: " ", Right: " ",
		TopLeft: " ", TopRight: " ", BottomLeft: " ", BottomRight: " "}
)

// BlockBorder draws the border with full blocks.
func BlockBorder() Border { return blockBorder }

// OuterHalfBlockBorder draws the border with half blocks that sit on the outer
// edge of its cells, so the line is thin and the box looks solid from outside.
func OuterHalfBlockBorder() Border { return outerHalfBlockBorder }

// InnerHalfBlockBorder draws the border with half blocks that sit on the inner
// edge of its cells.
func InnerHalfBlockBorder() Border { return innerHalfBlockBorder }

// DashedBorder draws thin dashed lines with square corners.
func DashedBorder() Border { return dashedBorder }

// HeavyDashedBorder draws heavy dashed lines with heavy corners.
func HeavyDashedBorder() Border { return heavyDashedBorder }

// MarkdownBorder draws a border out of "-" and "|", the way a Markdown table
// is written, so a copied box pastes as one.
func MarkdownBorder() Border { return markdownBorder }

// HiddenBorder takes the space of a border and draws none of it: the box keeps
// the size it has with a border, so switching a border on, for a focus ring
// say, moves nothing.
func HiddenBorder() Border { return hiddenBorder }
