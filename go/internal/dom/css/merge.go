package css

import (
	"strings"
)

// CN (class name) merges multiple class name strings intelligently,
// resolving Tailwind CSS conflicts by keeping only the last occurrence
// of conflicting utilities.
//
// Similar to shadcn/ui's cn function, it combines clsx-like merging
// with tailwind-merge conflict resolution.
//
// Example:
//
//	CN("px-4 py-2", "bg-blue-500")
//	// → "px-4 py-2 bg-blue-500"
//
//	CN("px-4", "px-2")  // Conflict: both are padding-x
//	// → "px-2"
//
//	CN("hover:bg-blue-500", "hover:bg-red-500")  // Same variant
//	// → "hover:bg-red-500"
//
//	CN("rounded-md px-3", "px-4 text-base")  // px-4 overrides px-3
//	// → "rounded-md px-4 text-base"
func CN(classes ...string) string {
	if len(classes) == 0 {
		return ""
	}

	// Split all input into individual class tokens
	var allClasses []string
	for _, cls := range classes {
		cls = strings.TrimSpace(cls)
		if cls == "" {
			continue
		}
		// Split by whitespace
		tokens := strings.Fields(cls)
		allClasses = append(allClasses, tokens...)
	}

	if len(allClasses) == 0 {
		return ""
	}

	// Track conflicts: map of "variant:conflictGroup" -> class index
	conflictMap := make(map[string]int)

	// Store metadata for each class
	type classInfo struct {
		original      string
		variant       string
		baseClass     string
		conflictGroup string
		conflictKey   string
		index         int
	}

	classInfos := make([]classInfo, len(allClasses))

	// First pass: parse all classes and identify conflicts
	for i, class := range allClasses {
		variant, baseClass := splitVariantAndClass(class)
		conflictGroup := getConflictGroup(baseClass)

		info := classInfo{
			original:      class,
			variant:       variant,
			baseClass:     baseClass,
			conflictGroup: conflictGroup,
			index:         i,
		}

		// Build conflict key: variant:conflictGroup
		// This ensures hover:px-4 and px-4 are in different conflict spaces
		if conflictGroup != "" {
			if variant != "" {
				info.conflictKey = variant + ":" + conflictGroup
			} else {
				info.conflictKey = conflictGroup
			}
		}

		classInfos[i] = info

		// Track the last occurrence of each conflict group
		if info.conflictKey != "" {
			conflictMap[info.conflictKey] = i
		}
	}

	// Second pass: build result, keeping only last occurrence of conflicts
	var result []string
	seen := make(map[string]bool) // Track which conflict keys we've already added

	for i, info := range classInfos {
		// If this class has a conflict group
		if info.conflictKey != "" {
			// Only include it if this is the last occurrence
			if conflictMap[info.conflictKey] == i {
				result = append(result, info.original)
				seen[info.conflictKey] = true
			}
			// Otherwise skip it (conflict resolved by later occurrence)
		} else {
			// No conflict group - always include (but avoid exact duplicates)
			result = append(result, info.original)
		}
	}

	// Remove exact duplicates while preserving order
	final := make([]string, 0, len(result))
	seenExact := make(map[string]bool)
	for _, class := range result {
		if !seenExact[class] {
			final = append(final, class)
			seenExact[class] = true
		}
	}

	return strings.Join(final, " ")
}
