package upload

import (
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m, goleak.IgnoreTopFunction("github.com/tus/tusd/v2/pkg/handler.newDelayedContext.func1"))
}
