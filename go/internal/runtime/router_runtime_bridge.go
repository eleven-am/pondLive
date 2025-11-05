package runtime

func init() {
	RegisterNavProvider(func(sess *ComponentSession) NavUpdate {
		navs := navHistory(sess)
		if len(navs) == 0 {
			return NavUpdate{}
		}
		last := navs[len(navs)-1]
		clearNavHistory(sess)
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
