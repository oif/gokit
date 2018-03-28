package errors

import (
	"fmt"

	"github.com/oif/gokit/runtime"
)

type E interface {
	Error() string
	DeepCopy() E
	SetRender(render) E
	Is(func(), ...error) E

	Code() int
	Class() string
	Set(key string, val interface{}) E
	Get(key string) (interface{}, bool)

	is(E) bool
	identity() string
}

type context map[string]interface{}
type render func(E) string

// status should be a error short description like `NOT_FOUND`, `BAD_REQUEST`
type fundamental struct {
	code    int
	class   string
	context context
	render  render
	source  string
}

type setContextFn func(*fundamental)

func WithContext(key string, val interface{}) setContextFn {
	return func(f *fundamental) {
		f.context[key] = val
	}
}

func New(code int, class string, fns ...setContextFn) E {
	f := new(fundamental)
	f.source = fmt.Sprintf("%p", f)
	f.code = code
	f.class = class
	f.context = make(context)

	for _, fn := range fns {
		fn(f)
	}
	return f
}

func (f *fundamental) DeepCopy() E   { return &*f }
func (f *fundamental) Code() int     { return f.code }
func (f *fundamental) Class() string { return f.class }

func (f *fundamental) Error() string {
	if f.render != nil {
		return f.render(f)
	}
	return fmt.Sprintf("[%d_%s] %v", f.code, f.class, f.context)
}

func (f *fundamental) Set(key string, val interface{}) E {
	f.context[key] = val
	return f
}

func (f *fundamental) Get(key string) (interface{}, bool) {
	val, ok := f.context[key]
	return val, ok
}

func (f *fundamental) SetRender(r render) E {
	f.render = r
	return f
}

func (f *fundamental) identity() string {
	return f.source
}

func (f *fundamental) is(e E) bool {
	return f.identity() == e.identity()
}

func (f *fundamental) Is(fn func(), es ...error) E {
	defer runtime.HandleCrash()

	for _, e := range es {
		err, ok := e.(E)
		if !ok {
			continue
		}
		if err.is(f) {
			fn()
			return f
		}
	}
	return f
}

func Is(left error, right E) bool {
	casted, ok := left.(E)
	if !ok {
		return false
	}
	return casted.is(right)
}
