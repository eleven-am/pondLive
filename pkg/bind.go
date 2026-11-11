package pkg

import "github.com/eleven-am/pondlive/internal/work"

func Bind(v any, opts ...BindOpt) Item { return work.Bind(v, opts...) }
