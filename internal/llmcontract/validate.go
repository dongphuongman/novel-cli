package llmcontract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"slices"
	"strconv"
)

// ValidateJSON 校验原始 JSON 是否满足直接返回契约使用的 JSON Schema 子集。
// 该子集覆盖 object/array/string/integer/number/boolean/null、required、enum 和
// additionalProperties。未声明 additionalProperties 时遵循 JSON Schema 默认语义，
// 不额外拒绝未知字段。
func ValidateJSON(schema map[string]any, raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("JSON parsing failed: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("there is an extra value after the JSON")
		}
		return fmt.Errorf("invalid trailing content after the JSON: %w", err)
	}
	return validateValue(schema, value, "$")
}

func validateValue(schema map[string]any, value any, path string) error {
	types, err := schemaTypes(schema["type"])
	if err != nil {
		return fmt.Errorf("%s has an invalid contract: %w", path, err)
	}
	if value == nil && !slices.Contains(types, "null") {
		return fmt.Errorf("%s must be %s, but is null", path, joinTypes(types))
	} else if value != nil {
		actual := valueType(value)
		if len(types) > 0 && !slices.Contains(types, actual) && !(actual == "integer" && slices.Contains(types, "number")) {
			return fmt.Errorf("%s must be %s, but is %s", path, joinTypes(types), actual)
		}
	}

	if rawEnum, exists := schema["enum"]; exists {
		enum, err := enumValues(rawEnum)
		if err != nil {
			return fmt.Errorf("%s.enum has an invalid contract: %w", path, err)
		}
		if !enumContains(enum, value) {
			return fmt.Errorf("%s must be one of %v, but is %v", path, enum, value)
		}
	}
	if value == nil {
		return nil
	}

	switch typed := value.(type) {
	case map[string]any:
		properties, ok := schema["properties"].(map[string]any)
		if !ok {
			return fmt.Errorf("%s contract is missing properties", path)
		}
		required, err := requiredNames(schema["required"])
		if err != nil {
			return fmt.Errorf("%s.required has an invalid contract: %w", path, err)
		}
		for _, name := range required {
			if _, exists := typed[name]; !exists {
				return fmt.Errorf("%s.%s is a required field", path, name)
			}
		}
		if allowAdditional, declared := schema["additionalProperties"].(bool); declared && !allowAdditional {
			for name := range typed {
				if _, exists := properties[name]; !exists {
					return fmt.Errorf("%s.%s is not declared in the contract", path, name)
				}
			}
		}
		for name, child := range typed {
			childSchema, exists := properties[name]
			if !exists {
				continue
			}
			childMap, ok := childSchema.(map[string]any)
			if !ok {
				return fmt.Errorf("the contract for %s.%s is not an object", path, name)
			}
			if err := validateValue(childMap, child, path+"."+name); err != nil {
				return err
			}
		}
	case []any:
		itemSchema, ok := schema["items"].(map[string]any)
		if !ok {
			return fmt.Errorf("%s contract is missing items", path)
		}
		for i, item := range typed {
			if err := validateValue(itemSchema, item, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	}
	return nil
}

func schemaTypes(value any) ([]string, error) {
	switch typed := value.(type) {
	case nil:
		return nil, nil
	case string:
		return []string{typed}, nil
	case []string:
		return typed, nil
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("a type union must contain only strings")
			}
			out = append(out, text)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("type must be a string or an array of strings")
	}
}

func requiredNames(value any) ([]string, error) {
	if value == nil {
		return nil, nil
	}
	if names, ok := stringSlice(value); ok {
		return names, nil
	}
	return nil, fmt.Errorf("must be an array of strings")
}

func stringSlice(value any) ([]string, bool) {
	switch typed := value.(type) {
	case []string:
		return typed, true
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil, false
			}
			out = append(out, text)
		}
		return out, true
	default:
		return nil, false
	}
}

func enumValues(value any) ([]any, error) {
	switch typed := value.(type) {
	case []string:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = item
		}
		return out, nil
	case []any:
		for _, item := range typed {
			if item == nil {
				continue
			}
			if _, ok := item.(string); !ok {
				return nil, fmt.Errorf("only strings and null are supported")
			}
		}
		return typed, nil
	default:
		return nil, fmt.Errorf("must be an array")
	}
}

func enumContains(enum []any, value any) bool {
	for _, item := range enum {
		if item == nil && value == nil {
			return true
		}
		itemText, itemOK := item.(string)
		valueText, valueOK := value.(string)
		if itemOK && valueOK && itemText == valueText {
			return true
		}
	}
	return false
}

func valueType(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		return "object"
	case []any:
		return "array"
	case string:
		return "string"
	case bool:
		return "boolean"
	case json.Number:
		if number, err := strconv.ParseFloat(string(typed), 64); err == nil && !math.IsInf(number, 0) && math.Trunc(number) == number {
			return "integer"
		}
		return "number"
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%T", value)
	}
}

func joinTypes(types []string) string {
	if len(types) == 0 {
		return "a valid JSON value"
	}
	return fmt.Sprint(types)
}
