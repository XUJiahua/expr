package conf

import "reflect"

type Tag struct {
	Type          reflect.Type
	MapTypesTable TypesTable // exists only if Type.Kind() == reflect.Map
	Method        bool
	Ambiguous     bool
}

type TypesTable map[string]Tag

// CreateTypesTable creates types table for type checks during parsing.
// If struct is passed, all fields will be treated as variables,
// as well as all fields of embedded structs and struct itself.
//
// If map is passed, all items will be treated as variables
// (key as name, value as type).
func CreateTypesTable(i interface{}) TypesTable {
	if i == nil {
		return nil
	}

	types := make(TypesTable)
	v := reflect.ValueOf(i)
	t := reflect.TypeOf(i)

	d := t
	if t.Kind() == reflect.Ptr {
		d = t.Elem()
	}

	switch d.Kind() {
	case reflect.Struct:
		types = FieldsFromStruct(d)

		// Methods of struct should be gathered from original struct with pointer,
		// as methods maybe declared on pointer receiver. Also this method retrieves
		// all embedded structs methods as well, no need to recursion.
		for i := 0; i < t.NumMethod(); i++ {
			m := t.Method(i)
			types[m.Name] = Tag{Type: m.Type, Method: true}
		}

	case reflect.Map:
		for _, key := range v.MapKeys() {
			value := v.MapIndex(key)
			if key.Kind() == reflect.String && value.IsValid() && value.CanInterface() {
				valueType := reflect.TypeOf(value.Interface())
				var subTypes TypesTable = nil
				if valueType.Kind() == reflect.Map {
					// usecase: fieldInLevel1.fieldInLevel2, support fieldInLevel2 type
					subTypes = CreateTypesTable(value.Interface())
				}
				types[key.String()] = Tag{Type: valueType, MapTypesTable: subTypes}
			}
		}

		// A map may have method too.
		for i := 0; i < t.NumMethod(); i++ {
			m := t.Method(i)
			types[m.Name] = Tag{Type: m.Type, Method: true}
		}
	}

	return types
}

func FieldsFromStruct(t reflect.Type) TypesTable {
	types := make(TypesTable)
	t = dereference(t)
	if t == nil {
		return types
	}

	switch t.Kind() {
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)

			if f.Anonymous {
				for name, typ := range FieldsFromStruct(f.Type) {
					if _, ok := types[name]; ok {
						types[name] = Tag{Ambiguous: true}
					} else {
						types[name] = typ
					}
				}
			}

			types[f.Name] = Tag{Type: f.Type}
		}
	}

	return types
}

func dereference(t reflect.Type) reflect.Type {
	if t == nil {
		return nil
	}
	if t.Kind() == reflect.Ptr {
		t = dereference(t.Elem())
	}
	return t
}
