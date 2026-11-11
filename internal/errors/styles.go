package errors

import "github.com/eleven-am/pondlive/internal/work"

func overlayStyles() work.Item {
	return work.Styles(map[string]string{
		"position":    "fixed",
		"inset":       "0",
		"background":  "rgba(0, 0, 0, 0.85)",
		"z-index":     "99999",
		"overflow":    "auto",
		"padding":     "2rem",
		"font-family": "ui-monospace, monospace",
	})
}

func containerStyles() work.Item {
	return work.Styles(map[string]string{
		"max-width":     "900px",
		"margin":        "0 auto",
		"background":    "#1e1e1e",
		"border-radius": "8px",
		"overflow":      "hidden",
	})
}

func headerStyles() work.Item {
	return work.Styles(map[string]string{
		"background":  "#dc2626",
		"color":       "white",
		"padding":     "1rem 1.5rem",
		"font-size":   "1.25rem",
		"font-weight": "600",
		"display":     "flex",
		"align-items": "center",
		"gap":         "0.75rem",
	})
}

func countBadgeStyles() work.Item {
	return work.Styles(map[string]string{
		"background":    "rgba(255,255,255,0.2)",
		"padding":       "0.25rem 0.5rem",
		"border-radius": "4px",
		"font-size":     "0.875rem",
	})
}

func errorListStyles() work.Item {
	return work.Styles(map[string]string{"padding": "1rem"})
}

func errorItemStyles() work.Item {
	return work.Styles(map[string]string{
		"background":    "#2d2d2d",
		"border-radius": "6px",
		"padding":       "1rem",
		"margin-bottom": "1rem",
	})
}

func errorHeaderStyles() work.Item {
	return work.Styles(map[string]string{
		"display":       "flex",
		"gap":           "0.5rem",
		"margin-bottom": "0.5rem",
	})
}

func codeStyles() work.Item {
	return work.Styles(map[string]string{
		"background":    "#dc2626",
		"color":         "white",
		"padding":       "0.125rem 0.375rem",
		"border-radius": "3px",
		"font-size":     "0.75rem",
	})
}

func phaseStyles() work.Item {
	return work.Styles(map[string]string{
		"background":    "#4a5568",
		"color":         "white",
		"padding":       "0.125rem 0.375rem",
		"border-radius": "3px",
		"font-size":     "0.75rem",
	})
}

func messageStyles() work.Item {
	return work.Styles(map[string]string{
		"color":       "#f87171",
		"font-size":   "1rem",
		"font-weight": "500",
	})
}

func componentPathStyles() work.Item {
	return work.Styles(map[string]string{
		"color":      "#a0aec0",
		"font-size":  "0.875rem",
		"margin-top": "0.5rem",
	})
}

func stackStyles() work.Item {
	return work.Styles(map[string]string{
		"margin-top":  "1rem",
		"border-top":  "1px solid #4a5568",
		"padding-top": "1rem",
	})
}

func frameStyles() work.Item {
	return work.Styles(map[string]string{
		"margin-bottom": "0.5rem",
		"font-size":     "0.875rem",
	})
}

func funcNameStyles() work.Item {
	return work.Styles(map[string]string{
		"color":   "#63b3ed",
		"display": "block",
	})
}

func fileStyles() work.Item {
	return work.Styles(map[string]string{
		"color":     "#718096",
		"font-size": "0.75rem",
	})
}

func crashContainerStyles() work.Item {
	return work.Styles(map[string]string{
		"min-height":      "100vh",
		"display":         "flex",
		"flex-direction":  "column",
		"align-items":     "center",
		"justify-content": "center",
		"padding":         "2rem",
		"font-family":     "system-ui, -apple-system, sans-serif",
		"background":      "#f5f5f5",
		"text-align":      "center",
	})
}

func crashTitleStyles() work.Item {
	return work.Styles(map[string]string{
		"color":         "#333",
		"margin-bottom": "1rem",
		"font-size":     "1.5rem",
	})
}

func crashMessageStyles() work.Item {
	return work.Styles(map[string]string{
		"color":     "#666",
		"max-width": "400px",
	})
}

func warningPanelStyles() work.Item {
	return work.Styles(map[string]string{
		"position":      "fixed",
		"right":         "1rem",
		"bottom":        "1rem",
		"width":         "min(480px, calc(100vw - 2rem))",
		"max-height":    "50vh",
		"overflow":      "auto",
		"z-index":       "99998",
		"background":    "#1f1a0e",
		"border":        "1px solid #d97706",
		"border-radius": "8px",
		"font-family":   "ui-monospace, monospace",
		"box-shadow":    "0 8px 24px rgba(0, 0, 0, 0.4)",
	})
}

func warningHeaderStyles() work.Item {
	return work.Styles(map[string]string{
		"background":  "#d97706",
		"color":       "#1f1a0e",
		"padding":     "0.5rem 0.75rem",
		"font-weight": "600",
		"display":     "flex",
		"align-items": "center",
		"gap":         "0.5rem",
	})
}

func warningBadgeStyles() work.Item {
	return work.Styles(map[string]string{
		"background":    "rgba(31,26,14,0.2)",
		"padding":       "0 0.375rem",
		"border-radius": "4px",
		"font-size":     "0.75rem",
		"flex":          "1",
	})
}

func warningButtonStyles() work.Item {
	return work.Styles(map[string]string{
		"background":    "transparent",
		"color":         "inherit",
		"border":        "1px solid currentColor",
		"border-radius": "4px",
		"padding":       "0 0.5rem",
		"font-size":     "0.75rem",
		"cursor":        "pointer",
	})
}

func warningListStyles() work.Item {
	return work.Styles(map[string]string{"padding": "0.5rem"})
}

func warningItemStyles() work.Item {
	return work.Styles(map[string]string{
		"background":    "#2b2414",
		"border-radius": "6px",
		"padding":       "0.75rem",
		"margin-bottom": "0.5rem",
		"color":         "#fbbf24",
	})
}

func warningItemHeaderStyles() work.Item {
	return work.Styles(map[string]string{
		"display":       "flex",
		"gap":           "0.5rem",
		"align-items":   "center",
		"margin-bottom": "0.5rem",
	})
}

func warningCodeStyles() work.Item {
	return work.Styles(map[string]string{
		"background":    "#d97706",
		"color":         "#1f1a0e",
		"padding":       "0.125rem 0.375rem",
		"border-radius": "3px",
		"font-size":     "0.75rem",
	})
}

func warningComponentStyles() work.Item {
	return work.Styles(map[string]string{
		"color":     "#fcd34d",
		"font-size": "0.75rem",
		"flex":      "1",
	})
}

func warningMessageStyles() work.Item {
	return work.Styles(map[string]string{
		"color":     "#fde68a",
		"font-size": "0.875rem",
	})
}

func warningStackStyles() work.Item {
	return work.Styles(map[string]string{
		"margin":      "0.5rem 0 0",
		"color":       "#a8a29e",
		"font-size":   "0.75rem",
		"white-space": "pre-wrap",
		"overflow":    "auto",
	})
}
