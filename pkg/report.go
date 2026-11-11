package pkg

import (
	"github.com/eleven-am/pondlive/internal/runtime"
)

type ReportLevel = runtime.ReportLevel

var (
	ReportWarning = runtime.ReportWarning
	ReportError   = runtime.ReportError
)

func Report(ctx *Ctx, level ReportLevel, code, message string) {
	ctx.Report(level, code, message)
}
