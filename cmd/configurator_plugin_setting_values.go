//go:build !windows

package main

import "fmt"

// Helper functions to extract current setting values from the untyped map.
func getSettingString(settings map[string]interface{}, key string, defaultVal interface{}) string {
	if value, ok := settings[key]; ok {
		return fmt.Sprintf("%v", value)
	}
	if defaultVal != nil {
		return fmt.Sprintf("%v", defaultVal)
	}
	return ""
}

func getSettingBool(settings map[string]interface{}, key string, defaultVal interface{}) bool {
	if value, ok := settings[key]; ok {
		switch boolValue := value.(type) {
		case bool:
			return boolValue
		case string:
			return boolValue == "true"
		}
	}
	if defaultVal != nil {
		if boolValue, ok := defaultVal.(bool); ok {
			return boolValue
		}
	}
	return false
}

func getSettingStringList(settings map[string]interface{}, key string, defaultVal interface{}) []string {
	value, ok := settings[key]
	if !ok {
		if defaultVal != nil {
			return convertToStringList(defaultVal)
		}
		return nil
	}
	return convertToStringList(value)
}

func convertToStringList(value interface{}) []string {
	switch list := value.(type) {
	case []interface{}:
		result := make([]string, 0, len(list))
		for _, item := range list {
			result = append(result, fmt.Sprintf("%v", item))
		}
		return result
	case []string:
		return list
	}
	return nil
}

type kvPair struct {
	Key   string
	Value string
}

// getSettingKeyValueListStructured extracts key-value pairs using the actual JSON field names
// from the stored map objects.
func getSettingKeyValueListStructured(
	settings map[string]interface{}, key, keyFieldName, valueFieldName string,
) []kvPair {
	value, ok := settings[key]
	if !ok {
		return nil
	}

	list, ok := value.([]interface{})
	if !ok {
		return nil
	}

	result := make([]kvPair, 0, len(list))
	for _, item := range list {
		fields, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		result = append(result, kvPair{
			Key:   fmt.Sprintf("%v", fields[keyFieldName]),
			Value: fmt.Sprintf("%v", fields[valueFieldName]),
		})
	}
	return result
}
