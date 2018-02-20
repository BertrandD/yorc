// Code generated by go-enum
// DO NOT EDIT!

package rest

import (
	"fmt"
	"strings"
)

const (
	// MapEntryOperationAdd is a MapEntryOperation of type Add
	MapEntryOperationAdd MapEntryOperation = iota
	// MapEntryOperationRemove is a MapEntryOperation of type Remove
	MapEntryOperationRemove
)

const _MapEntryOperationName = "AddRemove"

var _MapEntryOperationMap = map[MapEntryOperation]string{
	0: _MapEntryOperationName[0:3],
	1: _MapEntryOperationName[3:9],
}

func (i MapEntryOperation) String() string {
	if str, ok := _MapEntryOperationMap[i]; ok {
		return str
	}
	return fmt.Sprintf("MapEntryOperation(%d)", i)
}

var _MapEntryOperationValue = map[string]MapEntryOperation{
	_MapEntryOperationName[0:3]:                  0,
	strings.ToLower(_MapEntryOperationName[0:3]): 0,
	_MapEntryOperationName[3:9]:                  1,
	strings.ToLower(_MapEntryOperationName[3:9]): 1,
}

// ParseMapEntryOperation attempts to convert a string to a MapEntryOperation
func ParseMapEntryOperation(name string) (MapEntryOperation, error) {
	if x, ok := _MapEntryOperationValue[name]; ok {
		return MapEntryOperation(x), nil
	}
	return MapEntryOperation(0), fmt.Errorf("%s is not a valid MapEntryOperation", name)
}