package avatar

// An accessory is something the body wears on top: ears, a bunny's ears, an
// antenna, a tuft of hair or a pair of horns. It is part of the body, drawn in
// the body's colour, so it adds to the outline and never to the colours a cell
// has to hold. Most names wear none; a name that does gets a body drawn a little
// smaller so the whole figure still fits the frame.

type accessory int

const (
	accNone accessory = iota
	accCat
	accBunny
	accAntenna
	accTuft
	accHorns
)

// accessories holds, per accessory, its name, the upper edge of its band in
// [0, 1), how much the body is drawn smaller to make room, and how far it
// reaches above the body as a share of the body's height, which moves the whole
// figure down by half of that so it stays centred.
var accessories = [...]struct {
	name   string
	upTo   float64
	shrink float64
	up     float64
}{
	accNone:    {"none", 0.52, 1, 0},
	accCat:     {"ears", 0.63, 0.86, 0.35},
	accBunny:   {"bunny", 0.71, 0.72, 0.75},
	accAntenna: {"antenna", 0.79, 0.8, 0.5},
	accTuft:    {"tuft", 0.89, 0.88, 0.25},
	accHorns:   {"horns", 1, 0.86, 0.35},
}

// pickAccessory is the accessory a name wears on a body of this silhouette.
// Only the plain bodies wear one: a sun, a cloud, a nub body or a droplet
// already has something on top.
func pickAccessory(t traits, kind shapeKind) accessory {
	switch kind {
	case shapeRound, shapeOrganic, shapeBoxy, shapeCapsule, shapeHexagon:
	default:
		return accNone
	}
	v := t.at("acc.kind")
	for a, s := range accessories {
		if v < s.upTo {
			return accessory(a)
		}
	}
	return accHorns
}

// parts returns the circles and outlines of the accessory on body b, whose top
// is where the accessory sits. Each overlaps the body a little, so the union
// has no seam.
func (a accessory) parts(t traits, b ellipse) ([]circle, []path) {
	top := b.cy - b.ry
	var petals []circle
	var extra []path
	switch a {
	case accCat:
		size := 0.34 * b.rx
		for _, s := range []float64{-1, 1} {
			extra = append(extra, polygon(b.cx+s*0.6*b.rx, top+0.12*b.ry, size, size, 3, 0.3, s*18))
		}
	case accBunny:
		for _, s := range []float64{-1, 1} {
			extra = append(extra, superellipse(b.cx+s*0.4*b.rx, top-0.25*b.ry, 0.17*b.rx, 0.5*b.ry, 2.3, s*10))
		}
	case accAntenna:
		x := b.cx + t.jitter("acc.x", 0.25)*b.rx
		extra = append(extra, box(x, top-0.1*b.ry, 0.035*b.rx, 0.22*b.ry))
		petals = append(petals, circle{x, top - 0.36*b.ry, 0.11 * b.rx})
	case accTuft:
		x := b.cx + t.jitter("acc.x", 0.2)*b.rx
		petals = append(petals,
			circle{x - 0.12*b.rx, top - 0.02*b.ry, 0.1 * b.rx},
			circle{x + 0.02*b.rx, top - 0.12*b.ry, 0.12 * b.rx},
			circle{x + 0.16*b.rx, top - 0.03*b.ry, 0.09 * b.rx})
	case accHorns:
		for _, s := range []float64{-1, 1} {
			extra = append(extra, polygon(b.cx+s*0.52*b.rx, top+0.1*b.ry, 0.16*b.rx, 0.34*b.rx, 3, 0.2, s*28))
		}
	}
	return petals, extra
}

// String is the accessory's name, such as "ears"; "none" for no accessory.
func (a accessory) String() string {
	if a < 0 || int(a) >= len(accessories) {
		return accessories[accNone].name
	}
	return accessories[a].name
}
