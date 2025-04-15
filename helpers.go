// SPDX-License-Identifier: MIT
package meminfo

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
)

type optionalUint64 struct {
	Present bool
	Value   uint64
}

func readFileVarsIntoStruct(filename string, parseLine func(line string) (string, uint64, error), rv reflect.Value) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	numFieldsLeft := reflect.Indirect(rv).NumField()

	// Parse line by line
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if numFieldsLeft == 0 {
			break
		}

		key, value, err := parseLine(scanner.Text())
		if err != nil {
			continue
		}

		// Check if the key is one of the fields in memVars and set the value accordingly
		for i := 0; i < reflect.Indirect(rv).NumField(); i++ {
			fieldName := reflect.Indirect(rv).Type().Field(i).Name
			field := reflect.Indirect(rv).Field(i)
			if key == fieldName {
				numFieldsLeft--
				if field.Kind() == reflect.Struct {
					field.Set(reflect.ValueOf(optionalUint64{Present: true, Value: value}))
				} else if field.Kind() == reflect.Uint64 {
					field.Set(reflect.ValueOf(value))
				} else {
					panic(fmt.Sprintf("field %s has invalid type", fieldName))
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

