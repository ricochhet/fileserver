package configutil

import (
	"github.com/ricochhet/fileserver/pkg/contextutil"
)

type Context struct {
	*contextutil.Context[Config]
}

func NewContext() *Context {
	return &Context{
		Context: &contextutil.Context[Config]{},
	}
}
