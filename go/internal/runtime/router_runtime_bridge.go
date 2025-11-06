package runtime

func init() {
	RegisterNavProvider(func(sess *ComponentSession) NavUpdate {
		last, ok := consumePendingNavigation(sess)
		if !ok {
			return NavUpdate{}
		}
		target := buildNavURL(last)
		update := NavUpdate{}
		if last.T == "replace" {
			update.Replace = target
		} else if last.T == "nav" {
			update.Push = target
		}
		return update
	})
}
