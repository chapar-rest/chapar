// Copyright ©2021 The star-tex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kpath

type strset struct {
	db map[string]struct{}
	ks []string
}

func newStrSet(vs ...string) strset {
	set := strset{
		db: make(map[string]struct{}, len(vs)),
		ks: make([]string, len(vs)),
	}
	for i, v := range vs {
		set.db[v] = struct{}{}
		set.ks[i] = v
	}
	return set
}

func (set strset) has(k string) bool {
	_, ok := set.db[k]
	return ok
}

var (
	strsets = map[string]strset{
		"tex": newStrSet(
			".tex",
			".sty", ".cls", ".fd", ".aux", ".bbl", ".def", ".clo", ".ldf",
		),
		"texpool":            newStrSet(".pool"),
		"TeX system sources": newStrSet(".dtx", ".ins"),

		"gf":   newStrSet(".gf"),
		"pk":   newStrSet(".pk"),
		"tfm":  newStrSet(".tfm"),
		"afm":  newStrSet(".afm"),
		"base": newStrSet(".base"),
		"bib":  newStrSet(".bib"),
		"bst":  newStrSet(".bst"),
		"cnf":  newStrSet(".cnf"),
		"fmt":  newStrSet(".fmt"),
		"mf":   newStrSet(".mf"),
		"mft":  newStrSet(".mft"),
		"mp":   newStrSet(".mp"),
		"ofm":  newStrSet(".ofm", ".tfm"),
		"vf":   newStrSet(".vf"),
		"lig":  newStrSet(".lig"),

		"enc files":      newStrSet(".enc"),
		"type1 fonts":    newStrSet(".pfa", ".pfb"),
		"truetype fonts": newStrSet(".ttf", ".ttc", ".TTF", ".TTC", ".dfont"),
		"type42 fonts":   newStrSet(".t42", ".T42"),
		"opentype fonts": newStrSet(".otf", ".OTF"),
	}
)
