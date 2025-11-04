package runtime

type Effect interface{ isEffect() }

type Focus struct{ Selector string }

type Toast struct{ Message string }

type Push struct{ URL string }

type Replace struct{ URL string }

type ScrollTop struct{}

func (Focus) isEffect()     {}
func (Toast) isEffect()     {}
func (Push) isEffect()      {}
func (Replace) isEffect()   {}
func (ScrollTop) isEffect() {}
