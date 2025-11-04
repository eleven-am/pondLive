package router

import (
	runtime "github.com/eleven-am/liveui/internal/runtime"
)

func init() {
	runtime.RegisterNavProvider(func(sess *runtime.ComponentSession) runtime.NavUpdate {
		navs := navHistory(sess)
		if len(navs) == 0 {
			return runtime.NavUpdate{}
		}
		last := navs[len(navs)-1]
		clearNavHistory(sess)
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
