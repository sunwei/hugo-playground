package texttemplate

import (
	"context"
	"fmt"
	"github.com/sunwei/hugo-playground/tpl/internal/go_templates/texttemplate/parse"
	"io"
	"reflect"
)

// Preparer prepares the template before execution.
type Preparer interface {
	Prepare() (*Template, error)
}

// Executer executes a given template.
type Executer interface {
	ExecuteWithContext(ctx context.Context, p Preparer, wr io.Writer, data any) error
}

// Export it so we can populate Hugo's func map with it, which makes it faster.
var GoFuncs = builtinFuncs()

func NewExecuter(helper ExecHelper) Executer {
	return &executer{helper: helper}
}

// ExecHelper allows some custom eval hooks.
type ExecHelper interface {
	Init(ctx context.Context, tmpl Preparer)
	GetFunc(ctx context.Context, tmpl Preparer, name string) (reflect.Value, reflect.Value, bool)
	GetMethod(ctx context.Context, tmpl Preparer, receiver reflect.Value, name string) (method reflect.Value, firstArg reflect.Value)
	GetMapValue(ctx context.Context, tmpl Preparer, receiver, key reflect.Value) (reflect.Value, bool)
}

type executer struct {
	helper ExecHelper
}

type (
	dataContextKeyType string
)

const (
	// DataContextKey The data object passed to Execute or ExecuteWithContext gets stored with this key if not already set.
	DataContextKey = dataContextKeyType("data")
)

// ExecuteWithContext Note: The context is currently not fully implemeted in Hugo. This is a work in progress.
func (t *executer) ExecuteWithContext(ctx context.Context, p Preparer, wr io.Writer, data any) error {
	tmpl, err := p.Prepare()
	if err != nil {
		return err
	}

	if v := ctx.Value(DataContextKey); v == nil {
		ctx = context.WithValue(ctx, DataContextKey, data)
	}

	value, ok := data.(reflect.Value)
	if !ok {
		value = reflect.ValueOf(data)
	}

	state := &state{
		ctx:    ctx,
		helper: t.helper,
		prep:   p,
		tmpl:   tmpl,
		wr:     wr,
		vars:   []variable{{"$", value}},
	}

	t.helper.Init(ctx, p)

	return tmpl.executeWithState(state, value)
}

// state represents the state of an execution. It's not part of the
// template so that multiple executions of the same template
// can execute in parallel.
type state struct {
	tmpl   *Template
	ctx    context.Context // Added for Hugo. The orignal data context.
	prep   Preparer        // Added for Hugo.
	helper ExecHelper      // Added for Hugo.
	wr     io.Writer
	node   parse.Node // current node, for errors
	vars   []variable // push-down stack of variable values.
	depth  int        // the height of the stack of executing templates.
}

func (t *Template) executeWithState(state *state, value reflect.Value) (err error) {
	defer errRecover(&err)
	if t.Tree == nil || t.Root == nil {
		fmt.Printf("%q is an incomplete or empty template", t.Name())
	}
	state.walk(value, t.Root)
	return
}

// Prepare returns a template ready for execution.
func (t *Template) Prepare() (*Template, error) {
	return t, nil
}

// evalCall executes a function or method call. If it's a method, fun already has the receiver bound, so
// it looks just like a function call. The arg list, if non-nil, includes (in the manner of the shell), arg[0]
// as the function itself.
func (s *state) evalCall(dot, fun reflect.Value, isBuiltin bool, node parse.Node, name string, args []parse.Node, final reflect.Value, first ...reflect.Value) reflect.Value {
	if args != nil {
		args = args[1:] // Zeroth arg is function name/node; not passed to function.
	}

	typ := fun.Type()
	numFirst := len(first)
	numIn := len(args) + numFirst // Added for Hugo
	if final != missingVal {
		numIn++
	}
	numFixed := len(args) + len(first) // Adjusted for Hugo
	if typ.IsVariadic() {
		numFixed = typ.NumIn() - 1 // last arg is the variadic one.
		if numIn < numFixed {
			s.errorf("wrong number of args for %s: want at least %d got %d", name, typ.NumIn()-1, len(args))
		}
	} else if numIn != typ.NumIn() {
		s.errorf("wrong number of args for %s: want %d got %d", name, typ.NumIn(), numIn)
	}
	if !goodFunc(typ) {
		// TODO: This could still be a confusing error; maybe goodFunc should provide info.
		s.errorf("can't call method/function %q with %d results", name, typ.NumOut())
	}

	unwrap := func(v reflect.Value) reflect.Value {
		if v.Type() == reflectValueType {
			v = v.Interface().(reflect.Value)
		}
		return v
	}

	// Special case for builtin and/or, which short-circuit.
	if isBuiltin && (name == "and" || name == "or") {
		argType := typ.In(0)
		var v reflect.Value
		for _, arg := range args {
			v = s.evalArg(dot, argType, arg).Interface().(reflect.Value)
			if truth(v) == (name == "or") {
				// This value was already unwrapped
				// by the .Interface().(reflect.Value).
				return v
			}
		}
		if final != missingVal {
			// The last argument to and/or is coming from
			// the pipeline. We didn't short circuit on an earlier
			// argument, so we are going to return this one.
			// We don't have to evaluate final, but we do
			// have to check its type. Then, since we are
			// going to return it, we have to unwrap it.
			v = unwrap(s.validateType(final, argType))
		}
		return v
	}

	// Build the arg list.
	argv := make([]reflect.Value, numIn)
	// Args must be evaluated. Fixed args first.
	i := len(first)                                     // Adjusted for Hugo.
	for ; i < numFixed && i < len(args)+numFirst; i++ { // Adjusted for Hugo.
		argv[i] = s.evalArg(dot, typ.In(i), args[i-numFirst]) // Adjusted for Hugo.
	}
	// Now the ... args.
	if typ.IsVariadic() {
		argType := typ.In(typ.NumIn() - 1).Elem() // Argument is a slice.
		for ; i < len(args)+numFirst; i++ {       // Adjusted for Hugo.
			argv[i] = s.evalArg(dot, argType, args[i-numFirst]) // Adjusted for Hugo.
		}
	}
	// Add final value if necessary.
	if final != missingVal {
		t := typ.In(typ.NumIn() - 1)
		if typ.IsVariadic() {
			if numIn-1 < numFixed {
				// The added final argument corresponds to a fixed parameter of the function.
				// Validate against the type of the actual parameter.
				t = typ.In(numIn - 1)
			} else {
				// The added final argument corresponds to the variadic part.
				// Validate against the type of the elements of the variadic slice.
				t = t.Elem()
			}
		}
		argv[i] = s.validateType(final, t)
	}

	// Added for Hugo
	for i := 0; i < len(first); i++ {
		argv[i] = s.validateType(first[i], typ.In(i))
	}

	v, err := safeCall(fun, argv)
	// If we have an error that is not nil, stop execution and return that
	// error to the caller.
	if err != nil {
		s.at(node)
		s.errorf("error calling %s: %w", name, err)
	}
	return unwrap(v)
}
