package shape

import (
	"hash/fnv"

	"github.com/mirzakhany/yoga/render"
)

// LineCache caches shaped lines keyed by content hash.
type LineCache struct {
	shaper *Shaper
	data   map[uint64]Line
}

// NewLineCache returns a cache backed by shaper.
func NewLineCache(shaper *Shaper) *LineCache {
	return &LineCache{shaper: shaper, data: make(map[uint64]Line)}
}

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
	h := hashLineAt(text, logicalSize, false)
	if ln, ok := c.data[h]; ok {
		return ln
	}
	ln := c.shaper.ShapeLineAt(text, logicalSize)
	c.data[h] = ln
	return ln
}

// GetMonoAt shapes editor mono text at logicalSize, returning a cached line when possible.
func (c *LineCache) GetMonoAt(text string, logicalSize render.Px) Line {
	h := hashLineAt(text, logicalSize, true)
	if ln, ok := c.data[h]; ok {
		return ln
	}
	ln := c.shaper.ShapeLineMonoAt(text, logicalSize)
	c.data[h] = ln
	return ln
}

// Invalidate clears the cache (e.g. after font scale change).
func (c *LineCache) Invalidate() {
	c.data = make(map[uint64]Line)
}

func hashLineAt(s string, logicalSize float32, mono bool) uint64 {
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
	return h.Sum64()
}
