package minitracejs

import (
	"errors"

	"github.com/go-go-golems/go-minitrace/pkg/queryplan"
)

func (b *relationalBinding) buildError(code, path, message string) {
	err := b.vm.NewTypeError("%s", message)
	_ = err.Set("code", code)
	_ = err.Set("path", path)
	panic(err)
}

func (b *relationalBinding) checkBuild(err error, code, path string) {
	if err == nil {
		return
	}
	var scope *queryplan.ScopeError
	if errors.As(err, &scope) {
		code = "E_SCOPE"
	}
	b.buildError(code, path, err.Error())
}
