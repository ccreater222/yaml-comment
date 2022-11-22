package YamlComment

import (
	"errors"
	"gopkg.in/yaml.v3"
	"reflect"
	"strings"
)

const (
	nullTag      = "!!null"
	boolTag      = "!!bool"
	strTag       = "!!str"
	intTag       = "!!int"
	floatTag     = "!!float"
	timestampTag = "!!timestamp"
	seqTag       = "!!seq"
	mapTag       = "!!map"
	binaryTag    = "!!binary"
	mergeTag     = "!!merge"
)

var typeOfBytes = reflect.TypeOf([]byte(nil))

func Marshal(in interface{}) (out []byte, err error) {
	node, err := ToYamlNode(in)
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(node)
}

func ToYamlNode(data interface{}) (*yaml.Node, error) {
	node := yaml.Node{
		Kind:        0,
		Style:       0,
		Tag:         "",
		Value:       "",
		Anchor:      "",
		Alias:       nil,
		Content:     nil,
		HeadComment: "",
		LineComment: "",
		FootComment: "",
	}

	in := reflect.ValueOf(data)
	switch value := data.(type) {
	case reflect.Value:
		in = value
		var empty reflect.Value
		if in == empty {
			node.Encode(nil)
			return &node, nil
		}
		data = in.Interface()
	}
	switch value := data.(type) {
	case []byte:
		node.Encode(value)
		return &node, nil
	case nil:
		node.Encode(value)
		return &node, nil
	}
	switch in.Kind() {
	case reflect.Interface:
		return ToYamlNode(in.Elem())
	case reflect.Map:
		node.Kind = yaml.MappingNode
		node.Tag = mapTag
		node.Content = []*yaml.Node{}
		keys := in.MapKeys()
		for _, k := range keys {
			key_child_node, err := ToYamlNode(k)
			if err != nil {
				return nil, err
			}
			node.Content = append(node.Content, key_child_node)
			value_child_node, err := ToYamlNode(in.MapIndex(k))
			if err != nil {
				return nil, err
			}
			node.Content = append(node.Content, value_child_node)
		}
		return &node, nil
	case reflect.Ptr:
		return ToYamlNode(in.Elem())
	case reflect.Struct:
		node.Kind = yaml.MappingNode
		node.Tag = mapTag
		st := in.Type()
		n := st.NumField()
		for i := 0; i != n; i++ {
			field := st.Field(i)
			if field.PkgPath != "" && !field.Anonymous {
				continue // Private field
			}
			tag := field.Tag.Get("yaml")
			if tag == "" && strings.Index(string(field.Tag), ":") < 0 {
				tag = string(field.Tag)
			}
			if tag == "-" {
				continue
			}
			inline := false
			omit_empty := false
			fields := strings.Split(tag, ",")
			if len(fields) > 1 {
				for _, flag := range fields[1:] {
					switch flag {
					case "omitempty":
						omit_empty = true
					case "flow":
						return nil, errors.New("not implement flow command")
					case "inline":
						inline = true
					default:
						break
					}
				}
				tag = fields[0]
			}
			value := in.Field(i)
			if omit_empty {
				if isZero(value) {
					continue
				}
			}
			if inline {
				switch field.Type.Kind() {
				case reflect.Map:
					child_node, err := ToYamlNode(in.Field(i))
					if err != nil {
						return nil, err
					}
					node.Content = append(node.Content, child_node.Content...)

				case reflect.Struct, reflect.Ptr:
					ftype := field.Type
					for ftype.Kind() == reflect.Ptr {
						ftype = ftype.Elem()
					}
					if ftype.Kind() != reflect.Struct {
						return nil, errors.New("option ,inline may only be used on a struct or map field")
					}

					child_node, err := ToYamlNode(in.Field(i))
					if err != nil {
						return nil, err
					}
					node.Content = append(node.Content, child_node.Content...)

				default:
					return nil, errors.New("option ,inline may only be used on a struct or map field")
				}
				continue
			} else {
				if tag == "" {
					tag = strings.ToLower(field.Name)
				}
				child_node, err := ToYamlNode(tag)
				if err != nil {
					return nil, err
				}
				node.Content = append(node.Content, child_node)
				value_child_node, err := ToYamlNode(value)
				// add comment
				head_comment := field.Tag.Get("head_comment")
				foot_comment := field.Tag.Get("foot_comment")
				line_comment := field.Tag.Get("line_comment")
				value_child_node.HeadComment = head_comment
				value_child_node.FootComment = foot_comment
				value_child_node.LineComment = line_comment
				node.Content = append(node.Content, value_child_node)

			}
		}
		return &node, nil
	case reflect.Slice, reflect.Array:
		if in.Type() == typeOfBytes {
			node.Encode(in.Bytes())
			return &node, nil
		}
		node.Kind = yaml.SequenceNode
		node.Tag = seqTag
		n := in.Len()
		for i := 0; i < n; i++ {
			child_node, err := ToYamlNode(in.Index(i))
			if err != nil {
				return nil, err
			}
			node.Content = append(node.Content, child_node)
		}
		return &node, nil
	case reflect.String:
		node.Encode(in.String())
		return &node, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		node.Encode(in.Int())
		return &node, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		node.Encode(in.Uint())
		return &node, nil
	case reflect.Float32, reflect.Float64:
		node.Encode(in.Float())
		return &node, nil
	case reflect.Bool:
		node.Encode(in.Bool())
		return &node, nil
	default:
		panic("cannot marshal type: " + in.Type().String())
	}
	return &node, nil
}

func isZero(v reflect.Value) bool {
	kind := v.Kind()
	if z, ok := v.Interface().(yaml.IsZeroer); ok {
		if (kind == reflect.Ptr || kind == reflect.Interface) && v.IsNil() {
			return true
		}
		return z.IsZero()
	}
	switch kind {
	case reflect.String:
		return len(v.String()) == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	case reflect.Slice:
		return v.Len() == 0
	case reflect.Map:
		return v.Len() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Struct:
		vt := v.Type()
		for i := v.NumField() - 1; i >= 0; i-- {
			if vt.Field(i).PkgPath != "" {
				continue // Private field
			}
			if !isZero(v.Field(i)) {
				return false
			}
		}
		return true
	}
	return false
}
