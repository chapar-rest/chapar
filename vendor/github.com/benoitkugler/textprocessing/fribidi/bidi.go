package fribidi

import "fmt"

// This file implements most of Unicode Standard Annex #9, Tracking Number 13.

// pairingNode nodes are used for holding a pair of open/close brackets as
// described in BD16.
type pairingNode struct {
	open, close *oneRun
	next        *pairingNode
}

func (nodes *pairingNode) print() {
	fmt.Print("Pairs: ")
	for nodes != nil {
		fmt.Printf("(%d, %d) ", nodes.open.pos, nodes.close.pos)
		nodes = nodes.next
	}
	fmt.Println()
}

/* Search for an adjacent run in the forward or backward direction.
   It uses the next_isolate and prev_isolate run for short circuited searching.
*/

type stStack [bidiMaxResolvedLevels]struct {
	isolate      int
	override     CharType // only LTR, RTL and ON are valid
	level        Level
	isolateLevel Level
}

/* There are a few little points in pushing into and popping from the status
   stack:
   1. when the embedding level is not valid (more than
   FRIBIDI_BIDI_MAX_EXPLICIT_LEVEL=125), you must reject it, and not to push
   into the stack, but when you see a PDF, you must find the matching code,
   and if it was pushed in the stack, pop it, it means you must pop if and
   only if you have pushed the matching code, the over_pushed var counts the
   number of rejected codes so far.

   2. there's a more confusing point too, when the embedding level is exactly
   FRIBIDI_BIDI_MAX_EXPLICIT_LEVEL-1=124, an LRO, LRE, or LRI is rejected
   because the new level would be FRIBIDI_BIDI_MAX_EXPLICIT_LEVEL+1=126, that
   is invalid; but an RLO, RLE, or RLI is accepted because the new level is
   FRIBIDI_BIDI_MAX_EXPLICIT_LEVEL=125, that is valid, so the rejected codes
   may be not continuous in the logical order, in fact there are at most two
   continuous intervals of codes, with an RLO, RLE, or RLI between them.  To
   support this case, the first_interval var counts the number of rejected
   codes in the first interval, when it is 0, means that there is only one
   interval.

*/

/* a. If this new level would be valid, then this embedding code is valid.
   Remember (push) the current embedding level and override status.
   Reset current level to this new level, and reset the override status to
   newOverride.
   b. If the new level would not be valid, then this code is invalid. Don't
   change the current level or override status.
*/

// group the current state variable
type status struct {
	overPushed, firstInterval, stackSize *int
	level                                *Level
	override                             *CharType
}

func (st *stStack) pushStatus(isolateOverflow, isolate int, isolateLevel, newLevel Level, newOverride CharType, state status) {
	if *state.overPushed == 0 && isolateOverflow == 0 && newLevel <= maxExplicitLevel {
		if *state.level == maxExplicitLevel-1 {
			*state.firstInterval = *state.overPushed
		}
		st[*state.stackSize].level = *state.level
		st[*state.stackSize].isolateLevel = isolateLevel
		st[*state.stackSize].isolate = isolate
		st[*state.stackSize].override = *state.override
		*state.stackSize++
		*state.level = newLevel
		*state.override = newOverride
	} else if isolateOverflow == 0 {
		*state.overPushed++
	}
}

/* If there was a valid matching code, restore (pop) the last remembered
   (pushed) embedding level and directional override.
*/
func (st *stStack) popStatus(state status, isolate *int, isolateLevel *Level) {
	if *state.stackSize != 0 {
		if *state.overPushed > *state.firstInterval {
			*state.overPushed--
		} else {
			if *state.overPushed == *state.firstInterval {
				*state.firstInterval = 0
			}
			*state.stackSize--
			*state.level = (*st)[*state.stackSize].level
			*state.override = (*st)[*state.stackSize].override
			*isolate = (*st)[*state.stackSize].isolate
			*isolateLevel = (*st)[*state.stackSize].isolateLevel
		}
	}
}

// func getParDirection(bidiTypes []CharType) ParType {
// 	validIsolateCount := 0
// 	for _, bt := range bidiTypes {
// 		if bt == PDI {
// 			/* Ignore if there is no matching isolate */
// 			if validIsolateCount > 0 {
// 				validIsolateCount--
// 			}
// 		} else if bt.IsIsolate() {
// 			validIsolateCount++
// 		} else if validIsolateCount == 0 && bt.IsLetter() {
// 			if bt.IsRtl() {
// 				return RTL
// 			}
// 			return RTL
// 		}
// 	}
// 	return ON
// }

/* Push a new entry to the pairing linked list */
func (nodes *pairingNode) push(open, close *oneRun) *pairingNode {
	node := &pairingNode{}
	node.open = open
	node.close = close
	node.next = nodes
	return node
}

/* Sort by merge sort */
func (nodes *pairingNode) frontBackSplit(front **pairingNode, back **pairingNode) {
	//   PairingNode *pfast, *pslow;
	if nodes == nil || nodes.next == nil {
		*front = nodes
		*back = nil
	} else {
		pslow := nodes
		pfast := nodes.next
		for pfast != nil {
			pfast = pfast.next
			if pfast != nil {
				pfast = pfast.next
				pslow = pslow.next
			}
		}
		*front = nodes
		*back = pslow.next
		pslow.next = nil
	}
}

func sortedMerge(nodes1, nodes2 *pairingNode) *pairingNode {
	if nodes1 == nil {
		return nodes2
	}
	if nodes2 == nil {
		return nodes1
	}

	var res *pairingNode
	if nodes1.open.pos < nodes2.open.pos {
		res = nodes1
		res.next = sortedMerge(nodes1.next, nodes2)
	} else {
		res = nodes2
		res.next = sortedMerge(nodes1, nodes2.next)
	}
	return res
}

func sortPairingNodes(nodes **pairingNode) {
	/* 0 or 1 node case */
	if *nodes == nil || (*nodes).next == nil {
		return
	}

	var front, back *pairingNode
	(*nodes).frontBackSplit(&front, &back)
	sortPairingNodes(&front)
	sortPairingNodes(&back)
	*nodes = sortedMerge(front, back)
}

// GetParEmbeddingLevels finds the bidi embedding levels of a single paragraph,
// as defined by the Unicode Bidirectional Algorithm available at
// http://www.unicode.org/reports/tr9/.  This function implements rules P2 to
// I1 inclusive, and parts 1 to 3 of L1, except for rule X9 which is
// implemented in RemoveBidiMarks(). Part 4 of L1 is implemented
// in ReorderLine().
//
// `bidiTypes` is a list of bidi types as returned by GetBidiTypes()
// `bracketTypes` is either empty or a list of bracket types as returned by GetBracketTypes()
//
// Returns a slice of same length as `bidiTypes`, and the maximum level found plus one,
// which is thus always >= 1.
func GetParEmbeddingLevels(bidiTypes []CharType, bracketTypes []BracketType,
	pbaseDir *ParType) (embeddingLevels []Level, maxLevel Level) {
	if len(bidiTypes) == 0 {
		return nil, 1
	}
	var explicitsList, pp *oneRun

	// Determinate character types
	// Get run-length encoded character types
	mainRunList := encodeBidiTypes(bidiTypes, bracketTypes)

	/* Find base level */
	/* If no strong base_dir was found, resort to the weak direction
	   that was passed on input. */
	baseLevel := dirToLevel(*pbaseDir)
	if !pbaseDir.IsStrong() {
		/* P2. P3. Search for first strong character and use its direction as
		   base direction */
		validIsolateCount := 0
		for pp = mainRunList.next; pp.type_ != maskSentinel; pp = pp.next {
			if pp.type_ == PDI {
				/* Ignore if there is no matching isolate */
				if validIsolateCount > 0 {
					validIsolateCount--
				}
			} else if pp.type_.IsIsolate() {
				validIsolateCount++
			} else if validIsolateCount == 0 && pp.type_.IsLetter() {
				baseLevel = dirToLevel(pp.type_)
				*pbaseDir = levelToDir(baseLevel)
				break
			}
		}
	}
	baseDir := levelToDir(baseLevel)

	/* Explicit Levels and Directions */
	var (
		statusStack        stStack
		tempLink           oneRun
		prevIsolateLevel   Level /* When running over the isolate levels, remember the previous level */
		runPerIsolateLevel [bidiMaxResolvedLevels]*oneRun
	)

	/* explicits_list is a list like main_run_list, that holds the explicit
	   codes that are removed from main_run_list, to reinsert them later by
	   calling the shadow_run_list.
	*/
	explicitsList = newRunList()

	/* X1. Begin by setting the current embedding level to the paragraph
	   embedding level. Set the directional override status to neutral,
	   and directional isolate status to false.

	   Process each character iteratively, applying rules X2 through X8.
	   Only embedding levels from 0 to 123 are valid in this phase. */

	var (
		level = baseLevel
		/* stack */
		stackSize, overPushed, firstInterval int
		validIsolateCount, isolateOverflow   int
		override                             CharType = ON
		isolateLevel                         Level
		isolate                              int
		newOverride                          CharType
		newLevel                             Level
	)

	// used in push/pop operation
	vars := status{
		overPushed: &overPushed, firstInterval: &firstInterval, stackSize: &stackSize,
		level: &level, override: &override,
	}

	for pp = mainRunList.next; pp.type_ != maskSentinel; pp = pp.next {

		thisType := pp.type_
		pp.isolateLevel = isolateLevel

		if thisType.isExplicitOrBn() {
			if thisType.IsStrong() { /* LRE, RLE, LRO, RLO */
				/* 1. Explicit Embeddings */
				/*   X2. With each RLE, compute the least greater odd
				     embedding level. */
				/*   X3. With each LRE, compute the least greater even
				     embedding level. */
				/* 2. Explicit Overrides */
				/*   X4. With each RLO, compute the least greater odd
				     embedding level. */
				/*   X5. With each LRO, compute the least greater even
				     embedding level. */
				newOverride = explicitToOverrideDir(thisType)
				for i := pp.len; i != 0; i-- {
					newLevel = ((level + dirToLevel(thisType) + 2) & ^1) - dirToLevel(thisType)
					isolate = 0
					statusStack.pushStatus(isolateOverflow, isolate, isolateLevel, newLevel, newOverride, vars)
				}
			} else if thisType == PDF {
				/* 3. Terminating Embeddings and overrides */
				/*   X7. With each PDF, determine the matching embedding or
				     override code. */
				for i := pp.len; i != 0; i-- {
					if stackSize != 0 && statusStack[stackSize-1].isolate != 0 {
						break
					}
					statusStack.popStatus(vars, &isolate, &isolateLevel)
				}
			}

			/* X9. Remove all RLE, LRE, RLO, LRO, PDF, and BN codes. */
			/* Remove element and add it to explicits_list */
			pp.level = levelSentinel
			tempLink.next = pp.next
			explicitsList.moveNodeBefore(pp)
			pp = &tempLink
		} else if thisType == PDI {
			/* X6a. pop the direction of the stack */
			for i := pp.len; i != 0; i-- {
				if isolateOverflow > 0 {
					isolateOverflow--
					pp.level = level
				} else if validIsolateCount > 0 {
					/* Pop away all LRE,RLE,LRO, RLO levels
					   from the stack, as these are implicitly
					   terminated by the PDI */
					for stackSize != 0 && statusStack[stackSize-1].isolate == 0 {
						statusStack.popStatus(vars, &isolate, &isolateLevel)
					}
					overPushed = 0 /* The PDI resets the overpushed! */
					statusStack.popStatus(vars, &isolate, &isolateLevel)
					isolateLevel--
					validIsolateCount--
					pp.level = level
					pp.isolateLevel = isolateLevel
				} else {
					/* Ignore isolated PDI's by turning them into ON's */
					pp.type_ = ON
					pp.level = level
				}
			}
		} else if thisType.IsIsolate() {
			/* TBD support RL_LEN > 1 */
			newOverride = ON
			isolate = 1
			if thisType == LRI {
				newLevel = level + 2 - (level % 2)
			} else if thisType == RLI {
				newLevel = level + 1 + (level % 2)
			} else if thisType == FSI {
				/* Search for a local strong character until we
				   meet the corresponding PDI or the end of the
				   paragraph */
				//   Run *fsi_pp;
				isolateCount := 0
				var fsiBaseLevel Level
				for fsiPp := pp.next; fsiPp.type_ != maskSentinel; fsiPp = fsiPp.next {
					if fsiPp.type_ == PDI {
						isolateCount--
						if validIsolateCount < 0 {
							break
						}
					} else if fsiPp.type_.IsIsolate() {
						isolateCount++
					} else if isolateCount == 0 && fsiPp.type_.IsLetter() {
						fsiBaseLevel = dirToLevel(fsiPp.type_)
						break
					}
				}

				/* Same behavior like RLI and LRI above */
				if fsiBaseLevel.isRtl() != 0 {
					newLevel = level + 1 + (level % 2)
				} else {
					newLevel = level + 2 - (level % 2)
				}
			}

			pp.level = level
			pp.isolateLevel = isolateLevel
			if isolateLevel < maxExplicitLevel-1 {
				isolateLevel++
			}

			if !override.isNeutral() {
				pp.type_ = override
			}

			if newLevel <= maxExplicitLevel {
				validIsolateCount++
				statusStack.pushStatus(isolateOverflow, isolate, isolateLevel, newLevel, newOverride, vars)
				level = newLevel
			} else {
				isolateOverflow += 1
			}
		} else if thisType == BS {
			/* X8. All explicit directional embeddings and overrides are
			   completely terminated at the end of each paragraph. Paragraph
			   separators are not included in the embedding. */
			break
		} else {
			/* X6. For all types besides RLE, LRE, RLO, LRO, and PDF:
			   a. Set the level of the current character to the current
			   embedding level.
			   b. Whenever the directional override status is not neutral,
			   reset the current character type to the directional override
			   status. */
			pp.level = level
			if !override.isNeutral() {
				pp.type_ = override
			}
		}
	}

	/* Build the isolateLevel connections */
	prevIsolateLevel = 0
	for pp = mainRunList.next; pp.type_ != maskSentinel; pp = pp.next {
		isolateLevel := pp.isolateLevel

		/* When going from an upper to a lower level, zero out all higher levels
		   in order not erroneous connections! */
		if isolateLevel < prevIsolateLevel {
			for i := isolateLevel + 1; i <= prevIsolateLevel; i++ {
				runPerIsolateLevel[i] = nil
			}
		}
		prevIsolateLevel = isolateLevel

		if runPerIsolateLevel[isolateLevel] != nil {
			runPerIsolateLevel[isolateLevel].nextIsolate = pp
			pp.prevIsolate = runPerIsolateLevel[isolateLevel]
		}
		runPerIsolateLevel[isolateLevel] = pp
	}

	/* Implementing X8. It has no effect on a single paragraph! */
	level = baseLevel
	override = ON
	stackSize = 0
	overPushed = 0

	/* X10. The remaining rules are applied to each run of characters at the
	   same level. For each run, determine the start-of-level-run (sor) and
	   end-of-level-run (eor) type, either L or R. This depends on the
	   higher of the two levels on either side of the boundary (at the start
	   or end of the paragraph, the level of the 'other' run is the base
	   embedding level). If the higher level is odd, the type is R, otherwise
	   it is L. */
	/* Resolving Implicit Levels can be done out of X10 loop, so only change
	   of Resolving Weak Types and Resolving Neutral Types is needed. */

	mainRunList.compact()

	if debugMode {
		mainRunList.printTypesRe()
		mainRunList.printResolvedLevels()
		mainRunList.printResolvedTypes()
		fmt.Println("resolving weak types")
	}

	/* 4. Resolving weak types. Also calculate the maximum isolate level */
	var maxIsoLevel Level
	// int lastStrongStack[FRIBIDI_BIDI_MAX_RESOLVED_LEVELS];
	// CharType prev_type_orig;
	// bool w4;
	var lastStrongStack [bidiMaxResolvedLevels]CharType
	lastStrongStack[0] = baseDir

	for pp = mainRunList.next; pp.type_ != maskSentinel; pp = pp.next {

		pppPrev := pp.getAdjacentRun(false, false)
		pppNext := pp.getAdjacentRun(true, false)

		thisType := pp.type_
		isoLevel := pp.isolateLevel

		if isoLevel > maxIsoLevel {
			maxIsoLevel = isoLevel
		}

		var prevType, nextType CharType
		if pppPrev.level == pp.level {
			prevType = pppPrev.type_
		} else {
			prevType = levelToDir(maxL(pppPrev.level, pp.level))
		}

		if pppNext.level == pp.level {
			nextType = pppNext.type_
		} else {
			nextType = levelToDir(maxL(pppNext.level, pp.level))
		}

		if prevType.IsStrong() {
			lastStrongStack[isoLevel] = prevType
		}

		/* W1. NSM
		   Examine each non-spacing mark (NSM) in the level run, and change the
		   type of the NSM to the type of the previous character. If the NSM
		   is at the start of the level run, it will get the type of sor. */
		/* Implementation note: it is important that if the previous character
		   is not sor, then we should merge this run with the previous,
		   because of rules like W5, that we assume all of a sequence of
		   adjacent ETs are in one Run. */
		if thisType == NSM {
			/* New rule in Unicode 6.3 */
			if pp.prev.type_.IsIsolate() {
				pp.type_ = ON
			}

			if pppPrev.level == pp.level {
				if pppPrev == pp.prev {
					pp = pp.mergeWithPrev()
				}
			} else {
				pp.type_ = prevType
			}

			if prevType == nextType && pp.level == pp.next.level {
				if pppNext == pp.next {
					pp = pp.next.mergeWithPrev()
				}
			}
			continue /* As we know the next condition cannot be true. */
		}

		/* W2: European numbers. */
		if thisType == EN && lastStrongStack[isoLevel] == AL {
			pp.type_ = AN

			/* Resolving dependency of loops for rules W1 and W2, so we
			   can merge them in one loop. */
			if nextType == NSM {
				pppNext.type_ = AN
			}
		}
	}

	if debugMode {
		mainRunList.printResolvedLevels()
		mainRunList.printResolvedTypes()
		fmt.Println("4b. resolving weak types. W4 and W5")
	}

	/* The last iso level is used to invalidate the the last strong values when going from
	   a higher to a lower iso level. When this occur, all "last_strong" values are
	   set to the base_dir. */
	lastStrongStack[0] = baseDir

	/* Resolving dependency of loops for rules W4 and W5, W5 may
	   want to prevent W4 to take effect in the next turn, do this
	   through "w4". */
	w4 := true
	/* Resolving dependency of loops for rules W4 and W5 with W7,
	   W7 may change an EN to L but it sets the prevTypeOrig if needed,
	   so W4 and W5 in next turn can still do their works. */
	var prevTypeOrig CharType = ON

	/* Each isolate level has its own memory of the last strong character */
	for pp = mainRunList.next; pp.type_ != maskSentinel; pp = pp.next {

		thisType := pp.type_
		isoLevel := pp.isolateLevel

		pppPrev := pp.getAdjacentRun(false, false)
		pppNext := pp.getAdjacentRun(true, false)

		var prevType, nextType CharType
		if pppPrev.level == pp.level {
			prevType = pppPrev.type_
		} else {
			prevType = levelToDir(maxL(pppPrev.level, pp.level))
		}

		if pppNext.level == pp.level {
			nextType = pppNext.type_
		} else {
			nextType = levelToDir(maxL(pppNext.level, pp.level))
		}

		if prevType.IsStrong() {
			lastStrongStack[isoLevel] = prevType
		}

		/* W2 ??? */

		/* W3: Change ALs to R. */
		if thisType == AL {
			pp.type_ = RTL
			w4 = true
			prevTypeOrig = ON
			continue
		}

		/* W4. A single european separator changes to a european number.
		   A single common separator between two numbers of the same type
		   changes to that type. */
		if w4 && pp.len == 1 && thisType.isEsOrCs() &&
			prevTypeOrig.IsNumber() && prevTypeOrig == nextType &&
			(prevTypeOrig == EN || thisType == CS) {
			pp.type_ = prevType
			thisType = pp.type_
		}
		w4 = true

		/* W5. A sequence of European terminators adjacent to European
		   numbers changes to All European numbers. */
		if thisType == ET && (prevTypeOrig == EN || nextType == EN) {
			pp.type_ = EN
			w4 = false
			thisType = pp.type_
		}

		/* W6. Otherwise change separators and terminators to other neutral. */
		if thisType.isNumberSeparatorOrTerminator() {
			pp.type_ = ON
		}

		/* W7. Change european numbers to L. */
		if thisType == EN && lastStrongStack[isoLevel] == LTR {
			pp.type_ = LTR

			prevTypeOrig = ON
			if pp.level == pp.next.level {
				prevTypeOrig = EN
			}
		} else {
			prevTypeOrig = pp.next.prevTypeOrSOR()
		}
	}

	mainRunList.compactNeutrals()

	if debugMode {
		mainRunList.printResolvedLevels()
		mainRunList.printResolvedTypes()
		fmt.Println("5. Resolving Neutral Types - N0")
	}

	/*  BD16 - Build list of all pairs*/
	var (
		numIsoLevels      = int(maxIsoLevel + 1)
		pairingNodes      *pairingNode
		localBracketStack [localBracketSize][maxNestedBracketPairs]*oneRun
		bracketStack      [maxExplicitLevel][]*oneRun
		bracketStackSize  [maxExplicitLevel]int
		lastLevel         = mainRunList.level
		lastIsoLevel      Level
	)

	/* populate the bracket_size. The first LOCAL_BRACKET_SIZE entries
	   of the stack are on the stack. Allocate the rest of the entries.
	*/
	for isoLevel := 0; isoLevel < localBracketSize; isoLevel++ {
		bracketStack[isoLevel] = localBracketStack[isoLevel][:]
	}

	for isoLevel := localBracketSize; isoLevel < numIsoLevels; isoLevel++ {
		bracketStack[isoLevel] = make([]*oneRun, maxNestedBracketPairs)
	}

	/* Build the bd16 pair stack. */
	for pp = mainRunList.next; pp.type_ != maskSentinel; pp = pp.next {
		level := pp.level
		isoLevel := pp.isolateLevel
		brackProp := pp.bracketType

		/* Interpret the isolating run sequence as such that they
		   end at a change in the level, unless the iso_level has been
		   raised. */
		if level != lastLevel && lastIsoLevel == isoLevel {
			bracketStackSize[lastIsoLevel] = 0
		}

		if brackProp != NoBracket && pp.type_ == ON {
			if brackProp.isOpen() {
				if bracketStackSize[isoLevel] == maxNestedBracketPairs {
					break
				}

				/* push onto the pair stack */
				bracketStack[isoLevel][bracketStackSize[isoLevel]] = pp
				bracketStackSize[isoLevel]++
			} else {
				stackIdx := bracketStackSize[isoLevel] - 1
				for stackIdx >= 0 {
					seBrackProp := bracketStack[isoLevel][stackIdx].bracketType
					if seBrackProp.id() == brackProp.id() {
						bracketStackSize[isoLevel] = stackIdx

						pairingNodes = pairingNodes.push(bracketStack[isoLevel][stackIdx], pp)
						break
					}
					stackIdx--
				}
			}
		}
		lastLevel = level
		lastIsoLevel = isoLevel
	}

	/* The list must now be sorted for the next algo to work! */
	sortPairingNodes(&pairingNodes)

	if debugMode {
		pairingNodes.print()
	}

	/* Start the N0 */
	ppairs := pairingNodes
	for ppairs != nil {
		embeddingLevel := ppairs.open.level

		/* Find matching strong. */
		found := false
		var ppn *oneRun
		for ppn = ppairs.open; ppn != ppairs.close; ppn = ppn.next {
			thisType := ppn.typeAnEnAsRTL()

			/* Calculate level like in resolve implicit levels below to prevent
			   embedded levels not to match the base_level */
			thisLevel := ppn.level + (ppn.level.isRtl() ^ dirToLevel(thisType))

			/* N0b */
			if thisType.IsStrong() && thisLevel == embeddingLevel {
				var l CharType = LTR
				if thisLevel%2 != 0 {
					l = RTL
				}
				ppairs.close.type_ = l
				ppairs.open.type_ = l
				found = true
				break
			}
		}

		/* N0c */
		/* Search for any strong type preceding and within the bracket pair */
		if !found {
			/* Search for a preceding strong */
			precStrongLevel := embeddingLevel /* TBDov! Extract from Isolate level in effect */
			isoLevel := ppairs.open.isolateLevel
			for ppn = ppairs.open.prev; ppn.type_ != maskSentinel; ppn = ppn.prev {
				thisType := ppn.typeAnEnAsRTL()
				if thisType.IsStrong() && ppn.isolateLevel == isoLevel {
					precStrongLevel = ppn.level + (ppn.level.isRtl() ^ dirToLevel(thisType))
					break
				}
			}

			for ppn = ppairs.open; ppn != ppairs.close; ppn = ppn.next {
				thisType := ppn.typeAnEnAsRTL()
				if thisType.IsStrong() && ppn.isolateLevel == isoLevel {
					/* By constraint this is opposite the embedding direction,
					   since we did not match the N0b rule. We must now
					   compare with the preceding strong to establish whether
					   to apply N0c1 (opposite) or N0c2 embedding */
					var l CharType = LTR
					if precStrongLevel%2 != 0 {
						l = RTL
					}
					ppairs.open.type_ = l
					ppairs.close.type_ = l
					found = true
					break
				}
			}
		}

		ppairs = ppairs.next
	}

	/* Remove the bracket property and re-compact */
	for pp = mainRunList.next; pp.type_ != maskSentinel; pp = pp.next {
		pp.bracketType = NoBracket
	}
	mainRunList.compactNeutrals()

	if debugMode {
		mainRunList.printResolvedLevels()
		mainRunList.printResolvedTypes()
		fmt.Println("resolving neutral types - N1+N2")
	}

	// resolving neutral types - N1+N2
	for pp = mainRunList.next; pp.type_ != maskSentinel; pp = pp.next {

		pppPrev := pp.getAdjacentRun(false, false)
		pppNext := pp.getAdjacentRun(true, false)

		/* "European and Arabic numbers are treated as though they were R"
		FRIBIDI_CHANGE_NUMBER_TO_RTL does this. */
		thisType := pp.type_.changeNumberToRTL()

		var prevType, nextType CharType
		if pppPrev.level == pp.level {
			prevType = pppPrev.type_.changeNumberToRTL()
		} else {
			prevType = levelToDir(maxL(pppPrev.level, pp.level))
		}

		if pppNext.level == pp.level {
			nextType = pppNext.type_.changeNumberToRTL()
		} else {
			nextType = levelToDir(maxL(pppNext.level, pp.level))
		}

		if thisType.isNeutral() {
			if prevType == nextType {
				pp.type_ = prevType // N1
			} else {
				pp.type_ = pp.embeddingDirection() // N2
			}
		}
	}

	mainRunList.compact()

	if debugMode {
		mainRunList.printResolvedLevels()
		mainRunList.printResolvedTypes()
		fmt.Println("6. Resolving implicit levels")
	}

	maxLevel = baseLevel

	for pp = mainRunList.next; pp.type_ != maskSentinel; pp = pp.next {
		thisType := pp.type_
		level := pp.level

		/* I1. Even */
		/* I2. Odd */
		if thisType.IsNumber() {
			pp.level = (level + 2) & ^1
		} else {
			pp.level = level + (level.isRtl() ^ dirToLevel(thisType))
		}

		if pp.level > maxLevel {
			maxLevel = pp.level
		}
	}

	mainRunList.compact()

	if debugMode {
		fmt.Println(bidiTypes)
		mainRunList.printResolvedLevels()
		mainRunList.printResolvedTypes()
		fmt.Println("reinserting explicit codes")
	}

	/* Reinsert the explicit codes & BN's that are already removed, from the
	   explicits_list to main_run_list. */
	if explicitsList.next != explicitsList {
		//   register Run *p;
		shadowRunList(mainRunList, explicitsList, true)
		explicitsList = nil

		/* Set level of inserted explicit chars to that of their previous
		 * char, such that they do not affect reordering. */
		p := mainRunList.next
		if p != mainRunList && p.level == levelSentinel {
			p.level = baseLevel
		}
		for p = mainRunList.next; p.type_ != maskSentinel; p = p.next {
			if p.level == levelSentinel {
				p.level = p.prev.level
			}
		}
	}

	if debugMode {
		mainRunList.printTypesRe()
		mainRunList.printResolvedLevels()
		mainRunList.printResolvedTypes()
		fmt.Println("reset the embedding levels, 1, 2, 3.")
	}

	/* L1. Reset the embedding levels of some chars:
	   1. segment separators,
	   2. paragraph separators,
	   3. any sequence of whitespace characters preceding a segment
	      separator or paragraph separator, and
	   4. any sequence of whitespace characters and/or isolate formatting
	      characters at the end of the line.
	   ... (to be continued in ReorderLine()). */
	list := newRunList()
	q := list
	state := true
	pos := len(bidiTypes) - 1
	var (
		charType CharType
		p        *oneRun
	)
	for j := pos; j >= -1; j-- {
		/* close up the open link at the end */
		if j >= 0 {
			charType = bidiTypes[j]
		} else {
			charType = ON
		}
		if !state && charType.isSeparator() {
			state = true
			pos = j
		} else if state && !(charType.isExplicitOrSeparatorOrBnOrWs() || charType.IsIsolate()) {
			state = false
			p = &oneRun{}
			p.pos = j + 1
			p.len = pos - j
			p.type_ = baseDir
			p.level = baseLevel
			q.moveNodeBefore(p)
			q = p
		}
	}
	shadowRunList(mainRunList, list, false)

	if debugMode {
		mainRunList.printTypesRe()
		mainRunList.printResolvedLevels()
		mainRunList.printResolvedTypes()
		fmt.Println("leaving")
	}

	pos = 0
	embeddingLevels = make([]Level, len(bidiTypes))
	for pp = mainRunList.next; pp.type_ != maskSentinel; pp = pp.next {
		level := pp.level
		for l := pp.len; l != 0; l-- {
			embeddingLevels[pos] = level
			pos++
		}
	}

	return embeddingLevels, maxLevel + 1
}

func stringReverse(str []rune) {
	for i := len(str)/2 - 1; i >= 0; i-- {
		opp := len(str) - 1 - i
		str[i], str[opp] = str[opp], str[i]
	}
}

func indexesReverse(arr []int) {
	for i := len(arr)/2 - 1; i >= 0; i-- {
		opp := len(arr) - 1 - i
		arr[i], arr[opp] = arr[opp], arr[i]
	}
}

// ReorderLine reorders the characters in a line of text from logical to
// final visual order.  This function implements part 4 of rule L1, and rules
// L2 and L3 of the Unicode Bidirectional Algorithm available at
// http://www.unicode.org/reports/tr9/#Reordering_Resolved_Levels.
//
// As a side effect it also sets position maps if not nil.
//
// You should provide the resolved paragraph direction and embedding levels as
// set by GetParEmbeddingLevels(), which may change a bit.
// To be exact, the embedding level of any sequence of white space at the end of line
// is reset to the paragraph embedding level (according to part 4 of rule L1).
//
// Note that the bidi types and embedding levels are not reordered.  You can
// reorder these arrays using the map later.
//
// `visualStr` and `map_` must be either empty, or with same length as other inputs.
//
// See `Options` for more information.
//
// The maximum level found in this line plus one is returned
func ReorderLine(
	flags Options, bidiTypes []CharType,
	length, off int, // definition of the line in the paragraph
	baseDir ParType,
	/* input and output */
	embeddingLevels []Level, visualStr []rune, map_ []int) Level {
	var (
		maxLevel          Level
		hasVisual, hasMap = len(visualStr) != 0, len(map_) != 0
	)

	/* L1. Reset the embedding levels of some chars:
	   4. any sequence of white space characters at the end of the line. */
	for i := off + length - 1; i >= off && bidiTypes[i].isExplicitOrBnOrWs(); i-- {
		embeddingLevels[i] = dirToLevel(baseDir)
	}

	/* 7. Reordering resolved levels */
	var level Level

	/* Reorder both the outstring and the order array */
	if flags&ReorderNSM != 0 {
		/* L3. Reorder NSMs. */
		for i := off + length - 1; i >= off; i-- {
			if embeddingLevels[i].isRtl() != 0 && bidiTypes[i] == NSM {
				seqEnd := i
				level = embeddingLevels[i]

				for i--; i >= off && bidiTypes[i].isExplicitOrBnOrNsm() && embeddingLevels[i] == level; i-- {
				}

				if i < off || embeddingLevels[i] != level {
					i++
				}

				if hasVisual {
					stringReverse(visualStr[i : seqEnd+1])
				}
				if hasMap {
					indexesReverse(map_[i : seqEnd+1])
				}
			}
		}
	}

	/* Find max_level of the line.  We don't reuse the paragraph
	 * max_level, both for a cleaner API, and that the line max_level
	 * may be far less than paragraph max_level. */
	for i := off + length - 1; i >= off; i-- {
		if embeddingLevels[i] > maxLevel {
			maxLevel = embeddingLevels[i]
		}
	}
	/* L2. Reorder. */
	for level = maxLevel; level > 0; level-- {
		for i := off + length - 1; i >= off; i-- {
			if embeddingLevels[i] >= level {
				/* Find all stretches that are >= level_idx */
				seqEnd := i
				for i--; i >= off && embeddingLevels[i] >= level; i-- {
				}

				if hasVisual {
					stringReverse(visualStr[i+1 : seqEnd+1])
				}
				if hasMap {
					indexesReverse(map_[i+1 : seqEnd+1])
				}
			}
		}
	}

	return maxLevel + 1
}
