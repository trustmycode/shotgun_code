package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// SplitShotgunDiff parses a Git diff string and splits it into multiple
// smaller Git diff strings, each not exceeding approxLineLimit lines.
// It tries to split between file diffs first, then between hunks if a single file diff is too large.
func (a *App) SplitShotgunDiff(gitDiffText string, approxLineLimit int) ([]string, error) {
	safeLogInfof(a.ctx, "SplitShotgunDiff called with line limit: %d for git diff text", approxLineLimit)

	if strings.TrimSpace(gitDiffText) == "" {
		return []string{}, nil
	}

	fileDiffStartRegex := regexp.MustCompile(`(?m)^diff --git `)
	startIndices := fileDiffStartRegex.FindAllStringIndex(gitDiffText, -1)

	var fileDiffBlocks []string

	if len(startIndices) == 0 {
		safeLogWarning(a.ctx, fmt.Sprintf("SplitShotgunDiff: No 'diff --git' blocks found in input. Treating as single block."))
		if strings.TrimSpace(gitDiffText) != "" {
			fileDiffBlocks = append(fileDiffBlocks, gitDiffText)
		}
	} else {
		for i := 0; i < len(startIndices); i++ {
			start := startIndices[i][0]
			end := len(gitDiffText)
			if i+1 < len(startIndices) {
				end = startIndices[i+1][0]
			}
			block := gitDiffText[start:end]
			block = strings.TrimSpace(block)
			if block != "" {
				fileDiffBlocks = append(fileDiffBlocks, block)
			}
		}
	}

	var splitDiffs []string
	var currentSplitContent strings.Builder
	currentSplitLines := 0

	hunkHeaderRegex := regexp.MustCompile(`^@@ .* @@`)

	for _, fileBlock := range fileDiffBlocks {
		if fileBlock == "" {
			continue
		}

		fileBlockLines := strings.Split(fileBlock, "\n")
		numLinesInFileBlock := len(fileBlockLines)

		if numLinesInFileBlock > approxLineLimit {
			if currentSplitContent.Len() > 0 {
				splitDiffs = append(splitDiffs, currentSplitContent.String())
				currentSplitContent.Reset()
				currentSplitLines = 0
			}

			firstHunkIndex := -1
			for i, line := range fileBlockLines {
				if hunkHeaderRegex.MatchString(line) {
					firstHunkIndex = i
					break
				}
			}

			if firstHunkIndex == -1 {
				safeLogWarning(a.ctx, fmt.Sprintf("SplitShotgunDiff: Large file block without hunks in '%s'. Treating as single block.", getPathFromDiffHeader(fileBlockLines[0])))
				splitDiffs = append(splitDiffs, fileBlock+"\n")
				continue
			}

			fileHeader := strings.Join(fileBlockLines[:firstHunkIndex], "\n") + "\n"
			numLinesInHeader := firstHunkIndex

			var currentFileSplitHunks strings.Builder
			currentFileSplitHunkLines := 0

			hunkStartIndex := firstHunkIndex
			for hunkStartIndex < len(fileBlockLines) {
				hunkEndIndex := hunkStartIndex + 1
				for hunkEndIndex < len(fileBlockLines) && !hunkHeaderRegex.MatchString(fileBlockLines[hunkEndIndex]) {
					hunkEndIndex++
				}

				currentHunkContent := strings.Join(fileBlockLines[hunkStartIndex:hunkEndIndex], "\n")
				numLinesInCurrentHunk := hunkEndIndex - hunkStartIndex

				if numLinesInHeader+numLinesInCurrentHunk > approxLineLimit && currentFileSplitHunkLines == 0 {
					splitDiffs = append(splitDiffs, fileHeader+currentHunkContent+"\n")
					hunkStartIndex = hunkEndIndex
					continue
				}

				if currentFileSplitHunkLines > 0 && (numLinesInHeader+currentFileSplitHunkLines+numLinesInCurrentHunk > approxLineLimit) {
					splitDiffs = append(splitDiffs, fileHeader+currentFileSplitHunks.String())
					currentFileSplitHunks.Reset()
					currentFileSplitHunkLines = 0
				}

				currentFileSplitHunks.WriteString(currentHunkContent + "\n")
				currentFileSplitHunkLines += numLinesInCurrentHunk
				hunkStartIndex = hunkEndIndex
			}

			if currentFileSplitHunks.Len() > 0 {
				splitDiffs = append(splitDiffs, fileHeader+currentFileSplitHunks.String())
			}

		} else {
			if currentSplitLines > 0 && (currentSplitLines+numLinesInFileBlock > approxLineLimit) {
				splitDiffs = append(splitDiffs, currentSplitContent.String())
				currentSplitContent.Reset()
				currentSplitLines = 0
			}
			currentSplitContent.WriteString(fileBlock + "\n")
			currentSplitLines += numLinesInFileBlock
		}
	}

	if currentSplitContent.Len() > 0 {
		splitDiffs = append(splitDiffs, currentSplitContent.String())
	}

	initialSplitDiffs := make([]string, 0, len(splitDiffs))
	initialSplitSizes := make([]int, 0, len(splitDiffs))
	for _, sDiff := range splitDiffs {
		trimmedDiff := strings.TrimSpace(sDiff)
		if trimmedDiff != "" {
			initialSplitDiffs = append(initialSplitDiffs, trimmedDiff)
			initialSplitSizes = append(initialSplitSizes, len(strings.Split(trimmedDiff, "\n")))
		}
	}

	if approxLineLimit <= 0 {
		safeLogInfof(a.ctx, "approxLineLimit is %d, skipping merge step. Returning %d initial splits.", approxLineLimit, len(initialSplitDiffs))
		return initialSplitDiffs, nil
	}

	if len(initialSplitDiffs) <= 1 {
		safeLogInfof(a.ctx, "Only %d initial split(s), no merging needed. Returning as is.", len(initialSplitDiffs))
		return initialSplitDiffs, nil
	}

	safeLogInfof(a.ctx, "Starting advanced merge step for %d initial splits with approxLineLimit %d.", len(initialSplitDiffs), approxLineLimit)

	maxAllowedLines := int(float64(approxLineLimit) * 1.20)
	safeLogInfof(a.ctx, "Max allowed lines per merged split: %d", maxAllowedLines)

	type MergeGroup struct {
		Splits    []string
		LineCount int
	}

	var largeSplits []MergeGroup
	var smallSplits []int

	for i, size := range initialSplitSizes {
		if size >= approxLineLimit {
			largeSplits = append(largeSplits, MergeGroup{
				Splits:    []string{initialSplitDiffs[i]},
				LineCount: size,
			})
			safeLogInfof(a.ctx, "Split %d with %d lines kept as standalone group (already large)", i, size)
		} else {
			smallSplits = append(smallSplits, i)
		}
	}

	if len(smallSplits) == 0 {
		safeLogInfof(a.ctx, "No small splits to merge, returning %d large splits as-is", len(largeSplits))
		result := make([]string, len(largeSplits))
		for i, group := range largeSplits {
			result[i] = group.Splits[0]
		}
		return result, nil
	}

	smallSplitData := make([]struct {
		Content   string
		LineCount int
	}, len(smallSplits))

	for i, idx := range smallSplits {
		smallSplitData[i].Content = initialSplitDiffs[idx]
		smallSplitData[i].LineCount = initialSplitSizes[idx]
	}

	calculateSolutionScore := func(solution []MergeGroup) float64 {
		if len(solution) == 0 {
			return float64(1<<31 - 1)
		}

		score := float64(len(solution)) * 1000
		for _, group := range solution {
			utilization := float64(group.LineCount) / float64(maxAllowedLines)
			if utilization > 1.0 {
				score += 10000 * (utilization - 1.0)
			} else {
				score += 100 * (1.0 - utilization)
			}
		}

		return score
	}

	initialSolution := make([]MergeGroup, len(smallSplitData))
	for i, data := range smallSplitData {
		initialSolution[i] = MergeGroup{
			Splits:    []string{data.Content},
			LineCount: data.LineCount,
		}
	}

	currentSolution := initialSolution

	for {
		bestScore := calculateSolutionScore(currentSolution)
		var bestMerge struct {
			GroupIndex1 int
			GroupIndex2 int
			NewScore    float64
		}
		bestMerge.NewScore = bestScore
		mergeFound := false

		for i := 0; i < len(currentSolution); i++ {
			for j := i + 1; j < len(currentSolution); j++ {
				combinedLineCount := currentSolution[i].LineCount + currentSolution[j].LineCount + 1
				if combinedLineCount <= maxAllowedLines {
					newSolution := make([]MergeGroup, 0, len(currentSolution)-1)
					merged := MergeGroup{
						Splits:    append(append([]string{}, currentSolution[i].Splits...), currentSolution[j].Splits...),
						LineCount: combinedLineCount,
					}
					newSolution = append(newSolution, merged)

					for k := 0; k < len(currentSolution); k++ {
						if k != i && k != j {
							newSolution = append(newSolution, currentSolution[k])
						}
					}

					newScore := calculateSolutionScore(newSolution)
					if newScore < bestMerge.NewScore {
						bestMerge.GroupIndex1 = i
						bestMerge.GroupIndex2 = j
						bestMerge.NewScore = newScore
						mergeFound = true
					}
				}
			}
		}

		if !mergeFound || bestMerge.NewScore >= bestScore {
			break
		}

		i, j := bestMerge.GroupIndex1, bestMerge.GroupIndex2
		if i > j {
			i, j = j, i
		}

		combinedLineCount := currentSolution[i].LineCount + currentSolution[j].LineCount + 1
		currentSolution[i].Splits = append(currentSolution[i].Splits, currentSolution[j].Splits...)
		currentSolution[i].LineCount = combinedLineCount
		currentSolution = append(currentSolution[:j], currentSolution[j+1:]...)

		safeLogInfof(a.ctx, "Merged two groups, solution now has %d groups with score %.2f", len(currentSolution), bestMerge.NewScore)
	}

	finalGroups := append(largeSplits, currentSolution...)
	safeLogInfof(a.ctx, "Final solution: %d groups (%d large, %d optimized small groups)", len(finalGroups), len(largeSplits), len(currentSolution))

	mergedSplitsResult := make([]string, len(finalGroups))
	for i, group := range finalGroups {
		if len(group.Splits) == 1 {
			mergedSplitsResult[i] = group.Splits[0]
		} else {
			mergedSplitsResult[i] = strings.Join(group.Splits, "\n")
		}
		safeLogInfof(a.ctx, "Group %d: %d splits, %d lines", i, len(group.Splits), group.LineCount)
	}

	safeLogInfof(a.ctx, "Split git diff: %d initial splits, merged into %d final splits. Target line limit ~%d (merged max %d).", len(initialSplitDiffs), len(mergedSplitsResult), approxLineLimit, maxAllowedLines)
	return mergedSplitsResult, nil
}

func getPathFromDiffHeader(diffHeaderLine string) string {
	parts := strings.Fields(diffHeaderLine)
	if len(parts) >= 3 {
		return parts[2]
	}
	return "unknown_file"
}

func (a *App) StartupTest(ctx context.Context) {
	a.ctx = ctx
	a.contextGenerator = NewContextGenerator(a)
	a.fileWatcher = NewWatchman(a)
	a.settings.CustomIgnoreRules = defaultCustomIgnoreRulesContent
	a.settings.CustomPromptRules = defaultCustomPromptRulesContent
	_ = a.compileCustomIgnorePatterns()
}
