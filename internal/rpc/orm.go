package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/neko233-com/virtual-router-go/internal/core"
)

type RpcParamMeta struct {
	Name        string
	Description string
}

type RpcFuncMeta struct {
	PacketId    int
	Description string
	ParamMeta   []RpcParamMeta
	ClassName   string
	MethodName  string
}

func RegisterRpcFunc(meta RpcFuncMeta, fn any) error {
	if meta.PacketId <= 0 {
		return errors.New("packetId must be greater than 0")
	}
	if fn == nil {
		return errors.New("fn is nil")
	}
	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()
	if fnType.Kind() != reflect.Func {
		return errors.New("fn must be a function")
	}

	paramTypes := make([]reflect.Type, fnType.NumIn())
	for i := 0; i < fnType.NumIn(); i++ {
		paramTypes[i] = fnType.In(i)
	}

	className, methodName := meta.ClassName, meta.MethodName
	if className == "" || methodName == "" {
		fullName := runtime.FuncForPC(fnValue.Pointer()).Name()
		parts := strings.Split(fullName, "/")
		name := parts[len(parts)-1]
		if idx := strings.LastIndex(name, "."); idx >= 0 {
			className = name[:idx]
			methodName = name[idx+1:]
		} else {
			className = "go"
			methodName = name
		}
	}

	paramNames := make([]string, len(paramTypes))
	paramDescriptions := make([]string, len(paramTypes))
	for i := range paramTypes {
		if i < len(meta.ParamMeta) {
			paramNames[i] = fallback(meta.ParamMeta[i].Name, fmt.Sprintf("arg%d", i))
			paramDescriptions[i] = meta.ParamMeta[i].Description
		} else {
			paramNames[i] = fmt.Sprintf("arg%d", i)
			paramDescriptions[i] = ""
		}
	}

	parameterTypes := make([]string, len(paramTypes))
	parameterExampleJson := make([]string, len(paramTypes))
	for i, t := range paramTypes {
		parameterTypes[i] = typeName(t)
		parameterExampleJson[i] = exampleForType(t)
	}

	metaData := core.RpcStubMetadata{
		PacketId:              meta.PacketId,
		Description:           meta.Description,
		ClassName:             className,
		MethodName:            methodName,
		ParameterTypes:        parameterTypes,
		ParameterNames:        paramNames,
		ParameterDescriptions: paramDescriptions,
		ParameterExampleJson:  parameterExampleJson,
	}

	handler := func(args []json.RawMessage) (any, error) {
		if len(args) != len(paramTypes) {
			return nil, fmt.Errorf("参数数量不匹配: expected=%d actual=%d", len(paramTypes), len(args))
		}
		callArgs := make([]reflect.Value, len(paramTypes))
		for i, t := range paramTypes {
			val, err := unmarshalArgToType(args[i], t)
			if err != nil {
				return nil, fmt.Errorf("参数反序列化失败: index=%d type=%s err=%w", i, typeName(t), err)
			}
			callArgs[i] = val
		}

		out := fnValue.Call(callArgs)
		return normalizeFuncResult(out)
	}

	ServerStubManagerInstance().RegisterStub(metaData, handler)
	return nil
}

var errorType = reflect.TypeOf((*error)(nil)).Elem()

func normalizeFuncResult(out []reflect.Value) (any, error) {
	if len(out) == 0 {
		return nil, nil
	}
	if len(out) == 1 {
		if err, ok := out[0].Interface().(error); ok {
			return nil, err
		}
		return out[0].Interface(), nil
	}
	if len(out) == 2 {
		if !out[1].Type().Implements(errorType) {
			return nil, errors.New("rpc function second return value must be error")
		}
		if out[1].IsNil() {
			return out[0].Interface(), nil
		}
		return out[0].Interface(), out[1].Interface().(error)
	}
	return nil, errors.New("rpc function must return 0, 1 or 2 values")
}

func unmarshalArgToType(raw json.RawMessage, t reflect.Type) (reflect.Value, error) {
	if t.Kind() == reflect.Pointer {
		val := reflect.New(t.Elem())
		if err := json.Unmarshal(raw, val.Interface()); err != nil {
			return reflect.Value{}, err
		}
		return val, nil
	}
	val := reflect.New(t)
	if err := json.Unmarshal(raw, val.Interface()); err != nil {
		return reflect.Value{}, err
	}
	return val.Elem(), nil
}

func typeName(t reflect.Type) string {
	if t.Kind() == reflect.Pointer {
		return "*" + typeName(t.Elem())
	}
	if t.Name() != "" {
		return t.Name()
	}
	return t.String()
}

func exampleForType(t reflect.Type) string {
	if t.Kind() == reflect.Pointer {
		return exampleForType(t.Elem())
	}
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "0"
	case reflect.Float32, reflect.Float64:
		return "0.0"
	case reflect.Bool:
		return "false"
	case reflect.String:
		return "\"string_value\""
	case reflect.Slice, reflect.Array:
		return "[]"
	case reflect.Map:
		return "{}"
	case reflect.Struct:
		b, err := json.Marshal(reflect.New(t).Elem().Interface())
		if err == nil {
			return string(b)
		}
		return "{}"
	default:
		return "{}"
	}
}

func fallback(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}
