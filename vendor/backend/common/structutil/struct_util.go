package structutil

import "reflect"

func IsStructPtr(t reflect.Type) bool {
	return t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct
}

//check the item whether in object
//object must be slice
func HasItem(obj interface{}, item interface{}) bool {
	for i := 0; i < reflect.ValueOf(obj).Len(); i++ {
		if reflect.DeepEqual(reflect.ValueOf(obj).Index(i).Interface(), item) {
			return true
		}
	}
	return false
}

//拷贝结构体
//Be careful to use, from,to must be pointer
func DumpStruct(to interface{}, from interface{}) {
	fromv := reflect.ValueOf(from)
	tov := reflect.ValueOf(to)
	if fromv.Kind() != reflect.Ptr || tov.Kind() != reflect.Ptr {
		return
	}

	from_val := reflect.Indirect(fromv)
	to_val := reflect.Indirect(tov)

	for i := 0; i < from_val.Type().NumField(); i++ {
		fdi_from_val := from_val.Field(i)
		fd_name := from_val.Type().Field(i).Name
		fdi_to_val := to_val.FieldByName(fd_name)

		if fdi_to_val.IsValid() && fdi_from_val.Type() == fdi_to_val.Type() {
			fdi_to_val.Set(fdi_from_val)
		}
	}
}

//拷贝slice
func DumpList(to interface{}, from interface{}) {
	val_from := reflect.ValueOf(from)
	val_to := reflect.ValueOf(to)

	if val_from.Type().Kind() == reflect.Slice && val_to.Type().Kind() == reflect.Slice &&
		val_from.Len() == val_to.Len() {
		for i := 0; i < val_from.Len(); i++ {
			DumpStruct(val_to.Index(i).Addr().Interface(), val_from.Index(i).Addr().Interface())
		}
	}
}
