package gvcode

import (
	"bufio"
	"strings"
)

type TabStyle uint8

const (
	Tabs TabStyle = iota
	Spaces
)

// GuessIndentation guesses which kind of indentation the editor is
// using, returing the kind, if mixed indent is used, and the indent
// size in the case if spaces indentation.
func GuessIndentation(text string) (TabStyle, bool, int) {
	scanner := bufio.NewScanner(strings.NewReader(text))

	var tabs, spaces int
	var spaceWidths = make(map[int]int)

	indentScanner := func(batchSize int) bool {
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "" {
				continue // Ignore empty lines
			}

			// Count tabs and spaces
			if strings.HasPrefix(line, "\t") {
				tabs++
			} else if strings.HasPrefix(line, " ") {
				// Count leading spaces
				leading := len(line) - len(strings.TrimLeft(line, " "))
				spaces++
				spaceWidths[leading]++
			}

			// Stop early if we've analyzed enough lines
			if spaces+tabs > batchSize {
				return true
			}
		}

		return false
	}

	for  {
		hasMore := indentScanner(100)
		if (tabs + spaces<=5 || spaces == tabs) && hasMore {
			continue
		}

		mixedIndent := tabs>0 && spaces>0
		mainIndent := Tabs 
		if tabs > spaces {
			mainIndent = Tabs
		} else if spaces > tabs {
			mainIndent = Spaces
		}

		// If there are spaces, find the most common space width
		bestWidth, maxFreq := 4, 0
		for width, freq := range spaceWidths {
			if width > 0 && freq > maxFreq {
				bestWidth, maxFreq = width, freq
			}
		}

		return mainIndent, mixedIndent, bestWidth		
	}
}
