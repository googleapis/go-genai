// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apiform

import (
	"fmt"
	"io"
	"maps"
	"mime/multipart"
	"net/textproto"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/genai/interactions/internal/apijson/unmarshalinfo"
	"google.golang.org/genai/interactions/packages/apidata"
)

var encoders sync.Map // map[encoderEntry]encoderFunc

func Marshal(value any, writer *multipart.Writer) error {
	e := &encoder{
		dateFormat: time.RFC3339,
		arrayFmt:   "comma",
	}
	return e.marshal(value, writer)
}

// MarshalRoot marshals the multipart form
// MarshalRoot respects the [apidata.DynamicFields] pattern.
func MarshalRoot(value any, writer *multipart.Writer) error {
	e := &encoder{
		root:       true,
		dateFormat: time.RFC3339,
		arrayFmt:   "comma",
	}
	return e.marshal(value, writer)
}

func MarshalWithSettings(value any, writer *multipart.Writer, arrayFormat string) error {
	e := &encoder{
		arrayFmt:   arrayFormat,
		dateFormat: time.RFC3339,
	}
	return e.marshal(value, writer)
}

type encoder struct {
	arrayFmt   string
	dateFormat string
	root       bool
}

type encoderFunc func(key string, value reflect.Value, writer *multipart.Writer) error

type encoderField struct {
	tag parsedStructTag
	fn  encoderFunc
	idx []int
}

type encoderEntry struct {
	reflect.Type
	dateFormat string
	arrayFmt   string
	root       bool
}

func (e *encoder) marshal(value any, writer *multipart.Writer) error {
	val := reflect.ValueOf(value)
	if !val.IsValid() {
		return nil
	}
	typ := val.Type()
	enc := e.typeEncoder(typ)
	return enc("", val, writer)
}

func (e *encoder) typeEncoder(t reflect.Type) encoderFunc {
	entry := encoderEntry{
		Type:       t,
		dateFormat: e.dateFormat,
		arrayFmt:   e.arrayFmt,
		root:       e.root,
	}

	if fi, ok := encoders.Load(entry); ok {
		return fi.(encoderFunc)
	}

	// To deal with recursive types, populate the map with an
	// indirect func before we build it. This type waits on the
	// real func (f) to be ready and then calls it. This indirect
	// func is only used for recursive types.
	var (
		wg sync.WaitGroup
		f  encoderFunc
	)
	wg.Add(1)
	fi, loaded := encoders.LoadOrStore(entry, encoderFunc(func(key string, v reflect.Value, writer *multipart.Writer) error {
		wg.Wait()
		return f(key, v, writer)
	}))
	if loaded {
		return fi.(encoderFunc)
	}

	// Compute the real encoder and replace the indirect func with it.
	f = e.newTypeEncoder(t)
	wg.Done()
	encoders.Store(entry, f)
	return f
}

func (e *encoder) newTypeEncoder(t reflect.Type) encoderFunc {
	if t.ConvertibleTo(reflect.TypeOf(time.Time{})) {
		return e.newTimeTypeEncoder()
	}
	if t.Implements(reflect.TypeOf((*io.Reader)(nil)).Elem()) {
		return e.newReaderTypeEncoder()
	}
	e.root = false
	switch t.Kind() {
	case reflect.Pointer:
		inner := t.Elem()

		innerEncoder := e.typeEncoder(inner)
		return func(key string, v reflect.Value, writer *multipart.Writer) error {
			if !v.IsValid() || v.IsNil() {
				return nil
			}
			return innerEncoder(key, v.Elem(), writer)
		}
	case reflect.Struct:
		return e.newStructTypeEncoder(t)
	case reflect.Slice, reflect.Array:
		return e.newArrayTypeEncoder(t)
	case reflect.Map:
		return e.newMapEncoder(t)
	case reflect.Interface:
		return e.newInterfaceEncoder()
	default:
		return e.newPrimitiveTypeEncoder(t)
	}
}

func (e *encoder) newPrimitiveTypeEncoder(t reflect.Type) encoderFunc {
	switch t.Kind() {
	// Note that we could use `gjson` to encode these types but it would complicate our
	// code more and this current code shouldn't cause any issues
	case reflect.String:
		return func(key string, v reflect.Value, writer *multipart.Writer) error {
			return writer.WriteField(key, v.String())
		}
	case reflect.Bool:
		return func(key string, v reflect.Value, writer *multipart.Writer) error {
			if v.Bool() {
				return writer.WriteField(key, "true")
			}
			return writer.WriteField(key, "false")
		}
	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
		return func(key string, v reflect.Value, writer *multipart.Writer) error {
			return writer.WriteField(key, strconv.FormatInt(v.Int(), 10))
		}
	case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return func(key string, v reflect.Value, writer *multipart.Writer) error {
			return writer.WriteField(key, strconv.FormatUint(v.Uint(), 10))
		}
	case reflect.Float32:
		return func(key string, v reflect.Value, writer *multipart.Writer) error {
			return writer.WriteField(key, strconv.FormatFloat(v.Float(), 'f', -1, 32))
		}
	case reflect.Float64:
		return func(key string, v reflect.Value, writer *multipart.Writer) error {
			return writer.WriteField(key, strconv.FormatFloat(v.Float(), 'f', -1, 64))
		}
	default:
		return func(key string, v reflect.Value, writer *multipart.Writer) error {
			return fmt.Errorf("unknown type received at primitive encoder: %s", t.String())
		}
	}
}

func (e *encoder) newArrayTypeEncoder(t reflect.Type) encoderFunc {
	itemEncoder := e.typeEncoder(t.Elem())
	keyFn := e.arrayKeyEncoder()
	if e.arrayFmt == "comma" {
		return func(key string, v reflect.Value, writer *multipart.Writer) error {
			if v.Len() == 0 {
				return nil
			}
			elements := make([]string, v.Len())
			for i := 0; i < v.Len(); i++ {
				elements[i] = fmt.Sprint(v.Index(i).Interface())
			}
			return writer.WriteField(key, strings.Join(elements, ","))
		}
	}
	return func(key string, v reflect.Value, writer *multipart.Writer) error {
		if keyFn == nil {
			return fmt.Errorf("apiform: unsupported array format")
		}
		for i := 0; i < v.Len(); i++ {
			err := itemEncoder(keyFn(key, i), v.Index(i), writer)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

var extraFieldsType = reflect.TypeOf(apidata.DynamicFields(nil))

func (e *encoder) newStructTypeEncoder(t reflect.Type) encoderFunc {
	if unmarshalinfo.IsUnion(t) {
		return e.newStructUnionTypeEncoder(t)
	}

	encoderFields := []encoderField{}
	extraEncoder := (*encoderField)(nil)

	// This helper allows us to recursively collect field encoders into a flat
	// array. The parameter `index` keeps track of the access patterns necessary
	// to get to some field.
	var collectEncoderFields func(r reflect.Type, index []int)
	collectEncoderFields = func(r reflect.Type, index []int) {
		for i := 0; i < r.NumField(); i++ {
			idx := append(index, i)
			field := t.FieldByIndex(idx)
			if !field.IsExported() {
				continue
			}
			// If this is an embedded struct, traverse one level deeper to extract
			// the field and get their encoders as well.
			if field.Anonymous && field.Type.Kind() == reflect.Struct {
				collectEncoderFields(field.Type, idx)
				continue
			}
			// If json tag is not present, then we skip, which is intentionally
			// different behavior from the stdlib.
			ptag, ok := parseFormStructTag(field)
			if !ok {
				continue
			}
			// Inline fields encode their contents at the parent's key level
			// rather than nesting under a child key. There are two cases:
			//
			// 1. Root-level inline maps (additional properties):
			//
			//      type Params struct {
			//          Name   string         `form:"name"`
			//          Extras map[string]any `form:",inline"`
			//      }
			//
			//    The map entries are written as sibling form fields via
			//    encodeMapEntries after all struct fields. This uses the
			//    separate extraEncoder path and only applies at the root
			//    struct level (len(index) == 0) — an inline map inherited
			//    from an embedded struct falls through to case 2.
			//
			// 2. Inline structs, pointers, and non-root inline maps:
			//
			//      type Params struct {
			//          Name  string `form:"name"`
			//          Inner Inner  `form:",inline"`
			//      }
			//
			//    The field's type encoder is called with the parent's key,
			//    so its contents are encoded at the same level. Supports
			//    omitzero to skip the field entirely when zero-valued.
			if ptag.inline {
				ft := field.Type
				for ft.Kind() == reflect.Pointer {
					ft = ft.Elem()
				}
				// len(index) == 0: only the root struct's inline map gets the
				// dedicated extraEncoder path; an inline map inherited from an
				// embedded struct (depth > 0) falls through to the general case.
				if ft.Kind() == reflect.Map && len(index) == 0 {
					extraEncoder = &encoderField{ptag, e.typeEncoder(field.Type.Elem()), idx}
				} else {
					encoderFn := e.typeEncoder(field.Type)
					if ptag.omitzero {
						base := encoderFn
						encoderFn = func(key string, value reflect.Value, writer *multipart.Writer) error {
							if value.IsZero() {
								return nil
							}
							return base(key, value, writer)
						}
					}
					encoderFields = append(encoderFields, encoderField{ptag, encoderFn, idx})
				}
				continue
			}
			if ptag.name == "-" || ptag.name == "" {
				continue
			}

			dateFormat, ok := parseFormatStructTag(field)
			oldFormat := e.dateFormat
			if ok {
				switch dateFormat {
				case "date-time":
					e.dateFormat = time.RFC3339
				case "date":
					e.dateFormat = "2006-01-02"
				}
			}

			var encoderFn encoderFunc
			if ptag.omitzero {
				typeEncoderFn := e.typeEncoder(field.Type)
				encoderFn = func(key string, value reflect.Value, writer *multipart.Writer) error {
					if value.IsZero() {
						return nil
					}
					return typeEncoderFn(key, value, writer)
				}
			} else if ptag.defaultValue != nil {
				typeEncoderFn := e.typeEncoder(field.Type)
				encoderFn = func(key string, value reflect.Value, writer *multipart.Writer) error {
					if value.IsZero() {
						return typeEncoderFn(key, reflect.ValueOf(ptag.defaultValue), writer)
					}
					return typeEncoderFn(key, value, writer)
				}
			} else {
				encoderFn = e.typeEncoder(field.Type)
			}
			encoderFields = append(encoderFields, encoderField{ptag, encoderFn, idx})
			e.dateFormat = oldFormat
		}
	}
	collectEncoderFields(t, []int{})

	// Ensure deterministic output by sorting by lexicographic order
	sort.Slice(encoderFields, func(i, j int) bool {
		return encoderFields[i].tag.name < encoderFields[j].tag.name
	})

	extraFieldsIdx := unmarshalinfo.DynamicFieldsIndex(t)

	// Build a set of native field names for quick lookup when classifying extras.
	nativeNames := make(map[string]bool, len(encoderFields))
	for _, ef := range encoderFields {
		nativeNames[ef.tag.name] = true
	}

	return func(key string, value reflect.Value, writer *multipart.Writer) error {
		keyFn := e.objKeyEncoder(key)

		// Clone this so we can remove replacement fields and be left with only new fields at the end
		extrasFieldValue, _ := value.FieldByIndex(extraFieldsIdx).Interface().(apidata.DynamicFields)
		extras := maps.Clone(extrasFieldValue)
		for _, ef := range encoderFields {
			if extra, ok := extras[ef.tag.name]; ok {
				// This field has already been accounted for and doesn't need to be encoded later on
				delete(extras, ef.tag.name)
				if extra == apidata.Omit {
					continue
				}
				if err := writer.WriteField(keyFn(ef.tag.name), string(extra.(apidata.Unknown))); err != nil {
					return err
				}
			} else {
				field := value.FieldByIndex(ef.idx)
				if err := ef.fn(keyFn(ef.tag.name), field, writer); err != nil {
					return err
				}
			}
		}

		if err := e.encodeExtraFields(extras, keyFn, writer); err != nil {
			return err
		}

		if extraEncoder != nil {
			err := e.encodeMapEntries(key, value.FieldByIndex(extraEncoder.idx), writer)
			if err != nil {
				return err
			}
		}

		return nil
	}
}

// encodeExtraFields writes extra fields (from DynamicFields) to the multipart
// writer in sorted order for deterministic output.
func (e *encoder) encodeExtraFields(extras map[string]any, keyFn func(string) string, writer *multipart.Writer) error {
	if len(extras) == 0 {
		return nil
	}
	keys := make([]string, 0, len(extras))
	for k := range extras {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		val := extras[k]
		var err error
		if raw, ok := val.(apidata.Unknown); ok {
			err = writer.WriteField(keyFn(k), string(raw))
		} else if val != apidata.Omit {
			v := reflect.ValueOf(val)
			err = e.typeEncoder(v.Type())(keyFn(k), v, writer)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

var metadataType = reflect.TypeOf(unmarshalinfo.Metadata{})

func (e *encoder) newStructUnionTypeEncoder(t reflect.Type) encoderFunc {
	var fieldEncoders []encoderFunc
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Type == metadataType {
			fieldEncoders = append(fieldEncoders, nil)
		} else {
			fieldEncoders = append(fieldEncoders, e.typeEncoder(field.Type))
		}
	}

	return func(key string, value reflect.Value, writer *multipart.Writer) error {
		for i := 0; i < t.NumField(); i++ {
			if t.Field(i).Type == metadataType {
				continue
			}
			if !value.Field(i).IsZero() {
				return fieldEncoders[i](key, value.Field(i), writer)
			}
		}
		return fmt.Errorf("apiform: union %s has no field set", t.String())
	}
}

func (e *encoder) newTimeTypeEncoder() encoderFunc {
	format := e.dateFormat
	return func(key string, value reflect.Value, writer *multipart.Writer) error {
		return writer.WriteField(key, value.Convert(reflect.TypeOf(time.Time{})).Interface().(time.Time).Format(format))
	}
}

func (e encoder) newInterfaceEncoder() encoderFunc {
	return func(key string, value reflect.Value, writer *multipart.Writer) error {
		value = value.Elem()
		if !value.IsValid() {
			return nil
		}
		return e.typeEncoder(value.Type())(key, value, writer)
	}
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

func (e *encoder) newReaderTypeEncoder() encoderFunc {
	return func(key string, value reflect.Value, writer *multipart.Writer) error {
		reader, ok := value.Convert(reflect.TypeOf((*io.Reader)(nil)).Elem()).Interface().(io.Reader)
		if !ok {
			return nil
		}
		filename := "anonymous_file"
		contentType := "application/octet-stream"
		if named, ok := reader.(interface{ Filename() string }); ok {
			filename = named.Filename()
		} else if named, ok := reader.(interface{ Name() string }); ok {
			filename = path.Base(named.Name())
		}
		if typed, ok := reader.(interface{ ContentType() string }); ok {
			contentType = typed.ContentType()
		}

		// Below is taken almost 1-for-1 from [multipart.CreateFormFile]
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, escapeQuotes(key), escapeQuotes(filename)))
		h.Set("Content-Type", contentType)
		filewriter, err := writer.CreatePart(h)
		if err != nil {
			return err
		}
		_, err = io.Copy(filewriter, reader)
		return err
	}
}

func (e encoder) arrayKeyEncoder() func(string, int) string {
	var keyFn func(string, int) string
	switch e.arrayFmt {
	case "comma", "repeat":
		keyFn = func(k string, _ int) string { return k }
	case "brackets":
		keyFn = func(key string, _ int) string { return key + "[]" }
	case "indices:dots":
		keyFn = func(k string, i int) string {
			if k == "" {
				return strconv.Itoa(i)
			}
			return k + "." + strconv.Itoa(i)
		}
	case "indices:brackets":
		keyFn = func(k string, i int) string {
			if k == "" {
				return strconv.Itoa(i)
			}
			return k + "[" + strconv.Itoa(i) + "]"
		}
	}
	return keyFn
}

func (e encoder) objKeyEncoder(parent string) func(string) string {
	if parent == "" {
		return func(child string) string { return child }
	}
	switch e.arrayFmt {
	case "brackets":
		return func(child string) string { return parent + "[" + child + "]" }
	default:
		return func(child string) string { return parent + "." + child }
	}
}

// Given a []byte of json (may either be an empty object or an object that already contains entries)
// encode all of the entries in the map to the json byte array.
func (e *encoder) encodeMapEntries(key string, v reflect.Value, writer *multipart.Writer) error {
	type mapPair struct {
		key   string
		value reflect.Value
	}

	pairs := []mapPair{}

	iter := v.MapRange()
	for iter.Next() {
		if iter.Key().Type().Kind() == reflect.String {
			pairs = append(pairs, mapPair{key: iter.Key().String(), value: iter.Value()})
		} else {
			return fmt.Errorf("cannot encode a map with a non string key")
		}
	}

	// Ensure deterministic output
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].key < pairs[j].key
	})

	elementEncoder := e.typeEncoder(v.Type().Elem())
	keyFn := e.objKeyEncoder(key)
	for _, p := range pairs {
		err := elementEncoder(keyFn(p.key), p.value, writer)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *encoder) newMapEncoder(_ reflect.Type) encoderFunc {
	return func(key string, value reflect.Value, writer *multipart.Writer) error {
		return e.encodeMapEntries(key, value, writer)
	}
}
