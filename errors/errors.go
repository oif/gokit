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
	Status() string
	Set(key string, val interface{}) E
	Get(key string) (interface{}, bool)
}

type context map[string]interface{}
type render func(E) string

// code is a global unique code
// status should be a error short description like `NOT_FOUND`, `BAD_REQUEST`
type fundamental struct {
	code    int
	status  string
	context context
	render  render
	_source string
}

type setContextFn func(*fundamental)

func WithContext(key string, val interface{}) setContextFn {
	return func(f *fundamental) {
		f.context[key] = val
	}
}

func New(code int, status string, fns ...setContextFn) E {
	if RequireCodeUnique {
		if exist := codeBucket[code]; exist {
			panic(fmt.Sprintf("error code `%d` deplicate", code))
		}
		// Code not exist, then record it
		codeBucket[code] = true
	}
	f := new(fundamental)
	f._source = fmt.Sprintf("%p", f)
	f.code = code
	f.status = status
	f.context = make(context)

	for _, fn := range fns {
		fn(f)
	}
	return f
}

func (f *fundamental) Code() int      { return f.code }
func (f *fundamental) Status() string { return f.status }

func (f *fundamental) Error() string {
	if f.render != nil {
		return f.render(f)
	}
	return fmt.Sprintf("[%d_%s] %v", f.code, f.status, f.context)
}

func (f *fundamental) DeepCopy() E {
	return &*f
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

func (f *fundamental) Is(fn func(), es ...error) E {
	for _, e := range es {
		_e, ok := e.(*fundamental)
		if !ok {
			continue
		}
		if _e._source == f._source {
			defer runtime.HandleCrash()
			fn()
			return f
		}
	}
	return f
}
