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

	var allClasses []string
	for _, cls := range classes {
		cls = strings.TrimSpace(cls)
		if cls == "" {
			continue
		}

		tokens := strings.Fields(cls)
		allClasses = append(allClasses, tokens...)
	}

	if len(allClasses) == 0 {
		return ""
	}

	conflictMap := make(map[string]int)

	type classInfo struct {
		original      string
		variant       string
		baseClass     string
		conflictGroup string
		conflictKey   string
		index         int
	}

	classInfos := make([]classInfo, len(allClasses))

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

		if conflictGroup != "" {
			if variant != "" {
				info.conflictKey = variant + ":" + conflictGroup
			} else {
				info.conflictKey = conflictGroup
			}
		}

		classInfos[i] = info

		if info.conflictKey != "" {
			conflictMap[info.conflictKey] = i
		}
	}

	var result []string
	seen := make(map[string]bool)

	for i, info := range classInfos {

		if info.conflictKey != "" {

			if conflictMap[info.conflictKey] == i {
				result = append(result, info.original)
				seen[info.conflictKey] = true
			}

		} else {

			result = append(result, info.original)
		}
	}

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
