package css

import (
	"strings"
)

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
		important     bool
	}

	classInfos := make([]classInfo, len(allClasses))

	for i, class := range allClasses {
		important := strings.HasPrefix(class, "!")
		classForParse := strings.TrimPrefix(class, "!")

		variant, baseClass := splitVariantAndClass(classForParse)
		conflictGroup := getConflictGroup(baseClass)

		info := classInfo{
			original:      class,
			variant:       variant,
			baseClass:     baseClass,
			conflictGroup: conflictGroup,
			index:         i,
			important:     important,
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
			if prevIdx, ok := conflictMap[info.conflictKey]; ok {
				prev := classInfos[prevIdx]
				if prev.important && !info.important {
					continue
				}
			}
			conflictMap[info.conflictKey] = i
		}
	}

	var result []string
	seen := make(map[string]bool)

	for i, info := range classInfos {

		if info.conflictKey != "" {

			winnerIdx := conflictMap[info.conflictKey]
			winner := classInfos[winnerIdx]

			if winnerIdx == i || (winner.important && !info.important) {
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
