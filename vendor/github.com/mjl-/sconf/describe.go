package sconf

import (
	"bufio"
	"fmt"
	"reflect"
	"strings"
)

type writeError error

type writer struct {
	out    *bufio.Writer
	prefix string
}

func (w *writer) check(err error) {
	if err != nil {
		panic(writeError(err))
	}
}

func (w *writer) write(s string) {
	_, err := w.out.WriteString(s)
	w.check(err)
}

func (w *writer) flush() {
	err := w.out.Flush()
	w.check(err)
}

func (w *writer) indent() {
	w.prefix += "\t"
}

func (w *writer) unindent() {
	w.prefix = w.prefix[:len(w.prefix)-1]
}

func isOptional(sconfTag string) bool {
	l := strings.Split(sconfTag, ",")
	for _, s := range l {
		if s == "optional" {
			return true
		}
	}
	return false
}

func (w *writer) describeStruct(v reflect.Value) {
	t := v.Type()
	n := t.NumField()
	for i := 0; i < n; i++ {
		f := t.Field(i)
		doc := f.Tag.Get("sconf-doc")
		optional := isOptional(f.Tag.Get("sconf"))
		if doc != "" || optional {
			w.write(w.prefix)
			w.write("# ")
			w.write(doc)
			if optional {
				opt := "(optional)"
				if doc != "" {
					opt = " " + opt
				}
				w.write(opt)
			}
			w.write("\n")
		}
		w.write(w.prefix)
		w.write(f.Name + ":")
		w.describeValue(v.Field(i))
	}
}

func (w *writer) describeValue(v reflect.Value) {
	t := v.Type()
	i := v.Interface()
	switch t.Kind() {
	default:
		w.check(fmt.Errorf("unsupported value %v", t.Kind()))
		return

	case reflect.Bool:
		w.write(fmt.Sprintf(" %v\n", i))

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		w.write(fmt.Sprintf(" %d\n", i))

	case reflect.Float32, reflect.Float64:
		w.write(fmt.Sprintf(" %f\n", i))

	case reflect.String:
		w.write(fmt.Sprintf(" %s\n", i))

	case reflect.Slice:
		w.write("\n")
		w.indent()
		w.describeSlice(v)
		w.unindent()

	case reflect.Ptr:
		w.describeValue(reflect.New(t.Elem()).Elem())

	case reflect.Struct:
		w.write("\n")
		w.indent()
		w.describeStruct(v)
		w.unindent()
	}
}

func (w *writer) describeSlice(v reflect.Value) {
	n := v.Len()
	for i := 0; i < n; i++ {
		w.write(w.prefix)
		w.write("-")
		w.describeValue(v.Index(i))
	}
}
