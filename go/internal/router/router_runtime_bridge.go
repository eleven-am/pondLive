package router

import runtime "github.com/eleven-am/pondlive/go/internal/runtime"

func init() {
	runtime.RegisterNavProvider(func(sess *runtime.ComponentSession) runtime.NavUpdate {
		last, ok := consumePendingNavigation(sess)
		if !ok {
			return runtime.NavUpdate{}
		}
		target := buildNavURL(last)
		update := runtime.NavUpdate{}
		if last.T == "replace" {
			update.Replace = target
		} else if last.T == "nav" {
			update.Push = target
		}
		return update
	})
}
