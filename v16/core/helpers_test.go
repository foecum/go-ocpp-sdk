package core_test

import "reflect"

func reflectNew(t reflect.Type) any { return reflect.New(t).Interface() }
