package sconf

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"reflect"
)

// ParseFile reads an sconf file from path into dst.
func ParseFile(path string, dst interface{}) error {
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()
	return parse(path, src, dst)
}

// Parse reads an sconf file from a reader into dst.
func Parse(src io.Reader, dst interface{}) error {
	return parse("", src, dst)
}

// Describe writes a valid sconf file describing v to w.
func Describe(w io.Writer, v interface{}) (err error) {
	value := reflect.ValueOf(v)
	t := value.Type()
	if t.Kind() == reflect.Ptr {
		value = value.Elem()
		t = value.Type()
	}
	if t.Kind() != reflect.Struct {
		return fmt.Errorf("top level object must be a struct, is a %T", v)
	}
	defer func() {
		x := recover()
		if x == nil {
			return
		}
		if e, ok := x.(writeError); ok {
			err = error(e)
		} else {
			panic(x)
		}
	}()
	wr := &writer{out: bufio.NewWriter(w)}
	wr.describeStruct(value)
	wr.flush()
	return nil
}
