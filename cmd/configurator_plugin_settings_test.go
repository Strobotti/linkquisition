//go:build !windows

package main

import (
	"testing"

	"github.com/strobotti/linkquisition"
)

func testPluginMetadata() *linkquisition.PluginMetadata {
	return &linkquisition.PluginMetadata{
		Name: "TestPlugin",
		Settings: []linkquisition.PluginSettingDescriptor{
			{
				Key:     "stringVal",
				Type:    linkquisition.SettingTypeString,
				Default: "hello",
			},
			{
				Key:     "boolVal",
				Type:    linkquisition.SettingTypeBool,
				Default: true,
			},
			{
				Key:     "choiceVal",
				Type:    linkquisition.SettingTypeChoice,
				Default: "option1",
				Options: []string{"option1", "option2"},
			},
			{
				Key:  "noDefault",
				Type: linkquisition.SettingTypeString,
			},
			{
				Key:     "stringList",
				Type:    linkquisition.SettingTypeStringList,
				Default: []string{"a", "b", "c"},
			},
			{
				Key:     "duration",
				Type:    linkquisition.SettingTypeDuration,
				Default: "168h",
			},
		},
	}
}

func TestBuildDefaultSettings_ScalarTypes(t *testing.T) {
	t.Parallel()
	defaults := buildDefaultSettingsFromMetadata(testPluginMetadata())

	if v, ok := defaults["stringVal"]; !ok || v != "hello" {
		t.Errorf("expected stringVal = 'hello', got %v", v)
	}

	if v, ok := defaults["boolVal"]; !ok || v != true {
		t.Errorf("expected boolVal = true, got %v", v)
	}

	if v, ok := defaults["choiceVal"]; !ok || v != "option1" {
		t.Errorf("expected choiceVal = 'option1', got %v", v)
	}

	if v, ok := defaults["duration"]; !ok || v != "168h" {
		t.Errorf("expected duration = '168h', got %v", v)
	}
}

func TestBuildDefaultSettings_NoDefaultOmitted(t *testing.T) {
	t.Parallel()
	defaults := buildDefaultSettingsFromMetadata(testPluginMetadata())

	if _, ok := defaults["noDefault"]; ok {
		t.Error("expected noDefault to not be in the map")
	}
}

func TestBuildDefaultSettings_StringList(t *testing.T) {
	t.Parallel()
	defaults := buildDefaultSettingsFromMetadata(testPluginMetadata())

	v, ok := defaults["stringList"]
	if !ok {
		t.Fatal("expected stringList to be in the map")
	}

	list, ok := v.([]interface{})
	if !ok {
		t.Fatalf("expected stringList to be []interface{}, got %T", v)
	}

	if len(list) != 3 {
		t.Fatalf("expected 3 items, got %d", len(list))
	}

	if list[0] != "a" || list[1] != "b" || list[2] != "c" {
		t.Errorf("expected [a b c], got %v", list)
	}
}

func TestGetSettingStringList_WithDefault(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when no value and no default", func(t *testing.T) {
		t.Parallel()
		settings := map[string]interface{}{}
		result := getSettingStringList(settings, "key", nil)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("returns default when no value in settings", func(t *testing.T) {
		t.Parallel()
		settings := map[string]interface{}{}
		defaultVal := []string{"x", "y"}
		result := getSettingStringList(settings, "key", defaultVal)
		if len(result) != 2 || result[0] != "x" || result[1] != "y" {
			t.Errorf("expected [x y], got %v", result)
		}
	})

	t.Run("returns value from settings over default", func(t *testing.T) {
		t.Parallel()
		settings := map[string]interface{}{
			"key": []interface{}{"actual1", "actual2"},
		}
		defaultVal := []string{"default1"}
		result := getSettingStringList(settings, "key", defaultVal)
		if len(result) != 2 || result[0] != "actual1" || result[1] != "actual2" {
			t.Errorf("expected [actual1 actual2], got %v", result)
		}
	})
}
