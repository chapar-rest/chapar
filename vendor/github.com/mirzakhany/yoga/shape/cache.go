package shape

import (
	"hash/fnv"
	"unsafe"

	"github.com/mirzakhany/yoga/render"
)

// DefaultLineCacheBudget is the approximate byte budget for the hot generation
// of shaped lines. The cache holds at most two generations, so steady-state
// retention is roughly twice this number.
//
// A generation only has to cover the text touched between two evictions: one
// frame of UI chrome plus an editor viewport is a few hundred lines, so this
// leaves ample room for scrollback while keeping retention bounded. Without a
// budget the cache grew without limit — a scrolled-through 20k-line response
// body retained ~69 MB of shaped glyphs for the life of the process.
const DefaultLineCacheBudget = 8 << 20

// maxCachedGlyphs caps the glyph count of a cacheable line. A single very long
// line would otherwise consume a whole generation on its own; such lines are
// re-shaped on demand instead.
const maxCachedGlyphs = 4096

// LineCache caches shaped lines keyed by content hash.
//
// Eviction is generational rather than strict LRU: inserts land in hot, and
// when hot exceeds the byte budget it is demoted to cold and a fresh hot
// generation starts. A hit in cold is promoted back into hot, so lines still in
// use survive eviction while one-shot lines age out. This keeps a lookup to a
// map read with no per-hit bookkeeping, which matters because layout measures
// every visible string through this cache on every frame.
type LineCache struct {
	shaper *Shaper
	hot    map[uint64]Line
	cold   map[uint64]Line
	bytes  int // estimated bytes held by hot
	budget int
}

// NewLineCache returns a cache backed by shaper with the default byte budget.
func NewLineCache(shaper *Shaper) *LineCache {
	return &LineCache{
		shaper: shaper,
		hot:    make(map[uint64]Line),
		cold:   make(map[uint64]Line),
		budget: DefaultLineCacheBudget,
	}
}

// SetBudget sets the approximate byte budget for the hot generation. Values <= 0
// restore the default. A smaller budget takes effect at the next eviction.
func (c *LineCache) SetBudget(bytes int) {
	if bytes <= 0 {
		bytes = DefaultLineCacheBudget
	}
	c.budget = bytes
}

// Len reports the number of cached shaped lines across both generations.
func (c *LineCache) Len() int { return len(c.hot) + len(c.cold) }

// Get shapes UI text, returning a cached line when possible.
func (c *LineCache) Get(text string) Line {
	return c.GetAt(text, 0)
}

// GetMono shapes editor mono text, returning a cached line when possible.
func (c *LineCache) GetMono(text string) Line {
	return c.GetMonoAt(text, 0)
}

// GetAt shapes UI text at logicalSize, returning a cached line when possible.
func (c *LineCache) GetAt(text string, logicalSize render.Px) Line {
	return c.GetAtWeight(text, logicalSize, WeightRegular)
}

// GetAtWeight shapes UI text at logicalSize and weight.
func (c *LineCache) GetAtWeight(text string, logicalSize render.Px, weight int) Line {
	h := hashLineAt(text, logicalSize, false, weight)
	if ln, ok := c.lookup(h); ok {
		return ln
	}
	ln := c.shaper.ShapeLineAtWeight(text, logicalSize, weight)
	c.store(h, ln)
	return ln
}

// GetMonoAt shapes editor mono text at logicalSize, returning a cached line when possible.
func (c *LineCache) GetMonoAt(text string, logicalSize render.Px) Line {
	h := hashLineAt(text, logicalSize, true, WeightRegular)
	if ln, ok := c.lookup(h); ok {
		return ln
	}
	ln := c.shaper.ShapeLineMonoAt(text, logicalSize)
	c.store(h, ln)
	return ln
}

// lookup reads both generations, promoting a cold hit back into hot.
func (c *LineCache) lookup(h uint64) (Line, bool) {
	if ln, ok := c.hot[h]; ok {
		return ln, true
	}
	ln, ok := c.cold[h]
	if !ok {
		return Line{}, false
	}
	delete(c.cold, h)
	c.insertHot(h, ln)
	return ln, true
}

// store caches ln unless it is large enough to dominate a whole generation.
func (c *LineCache) store(h uint64, ln Line) {
	if len(ln.Glyphs) > maxCachedGlyphs {
		return
	}
	c.insertHot(h, ln)
}

func (c *LineCache) insertHot(h uint64, ln Line) {
	c.hot[h] = ln
	c.bytes += lineBytes(ln)
	if c.bytes > c.budget {
		// Demote hot to cold and start a fresh generation. Dropping the
		// previous cold generation is what bounds total retention.
		c.cold = c.hot
		c.hot = make(map[uint64]Line, len(c.cold)/2+1)
		c.bytes = 0
	}
}

// Invalidate clears the cache (e.g. after font scale change).
func (c *LineCache) Invalidate() {
	c.hot = make(map[uint64]Line)
	c.cold = make(map[uint64]Line)
	c.bytes = 0
}

const (
	glyphSize = int(unsafe.Sizeof(Glyph{}))
	// lineEntryOverhead approximates the map bucket plus the two slice headers
	// in Line. It only has to be the right order of magnitude for the budget to
	// bound retention.
	lineEntryOverhead = 96
)

// lineBytes estimates the retained size of a shaped line.
func lineBytes(ln Line) int {
	return lineEntryOverhead + len(ln.Glyphs)*glyphSize + len(ln.Runes)*4
}

func hashLineAt(s string, logicalSize float32, mono bool, weight int) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	if logicalSize != 0 {
		var buf [4]byte
		bits := uint32(logicalSize * 1000)
		buf[0] = byte(bits)
		buf[1] = byte(bits >> 8)
		buf[2] = byte(bits >> 16)
		buf[3] = byte(bits >> 24)
		_, _ = h.Write(buf[:])
	}
	if mono {
		_, _ = h.Write([]byte{1})
	}
	if weight >= WeightSemiBold {
		_, _ = h.Write([]byte{2})
	}
	return h.Sum64()
}
