package matcher

import (
	"github.com/simonalong/mikilin-go/constant"
	log "github.com/sirupsen/logrus"
	"reflect"
	"strings"
)

type CustomizeMatch struct {
	BlackWhiteMatch

	expression string
	funValue   reflect.Value
}

var funMap = make(map[string]interface{})

type MatchJudge func(interface{}) bool

func (customizeMatch *CustomizeMatch) Match(parameterMap map[string]interface{}, object interface{}, field reflect.StructField, fieldValue interface{}) bool {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("call match err: %v", err)
			return
		}
	}()

	var in []reflect.Value
	if customizeMatch.funValue.Type().NumIn() == 1 {
		in = make([]reflect.Value, 1)
		inKind0 := customizeMatch.funValue.Type().In(0).Kind()
		if inKind0 == reflect.ValueOf(object).Kind() {
			in[0] = reflect.ValueOf(object)
		} else if inKind0 == reflect.ValueOf(fieldValue).Kind() {
			in[0] = reflect.ValueOf(fieldValue)
		} else {
			log.Errorf("the value don't match parameter of fun")
			return false
		}
	} else if customizeMatch.funValue.Type().NumIn() == 2 {
		in = make([]reflect.Value, 2)
		inKind0 := customizeMatch.funValue.Type().In(0).Kind()
		inKind1 := customizeMatch.funValue.Type().In(1).Kind()

		if inKind0 == reflect.ValueOf(object).Kind() {
			in[0] = reflect.ValueOf(object)
			if inKind1 == reflect.ValueOf(fieldValue).Kind() {
				in[1] = reflect.ValueOf(fieldValue)
			} else if inKind1 == reflect.ValueOf(parameterMap).Kind() {
				in[1] = reflect.ValueOf(parameterMap)
			} else {
				log.Errorf("参数2 没有找到匹配的类型的值，只可为属性的类型或者map[string]interface{}的类型")
				return false
			}
		} else if inKind0 == reflect.ValueOf(fieldValue).Kind() {
			in[0] = reflect.ValueOf(fieldValue)
			if inKind1 == reflect.ValueOf(object).Kind() {
				in[1] = reflect.ValueOf(object)
			} else if inKind1 == reflect.ValueOf(parameterMap).Kind() {
				in[1] = reflect.ValueOf(parameterMap)
			} else {
				log.Errorf("参数2 没有找到匹配的类型的值，只可为属性所在的对象的类型或者map[string]interface{}的类型")
				return false
			}
		} else {
			log.Errorf("没有找到匹配的类型的值")
			return false
		}
	} else if customizeMatch.funValue.Type().NumIn() == 3 {
		in = make([]reflect.Value, 3)
		inKind0 := customizeMatch.funValue.Type().In(0).Kind()
		inKind1 := customizeMatch.funValue.Type().In(1).Kind()
		inKind2 := customizeMatch.funValue.Type().In(2).Kind()

		if inKind0 == reflect.ValueOf(object).Kind() && inKind1 == reflect.ValueOf(fieldValue).Kind() {
			in[0] = reflect.ValueOf(object)
			in[1] = reflect.ValueOf(fieldValue)
		} else if inKind0 == reflect.ValueOf(fieldValue).Kind() && inKind1 == reflect.ValueOf(object).Kind() {
			in[0] = reflect.ValueOf(fieldValue)
			in[1] = reflect.ValueOf(object)
		}

		if inKind2 == reflect.ValueOf(parameterMap).Kind() {
			in[2] = reflect.ValueOf(parameterMap)
		} else {
			log.Errorf("参数3不是map[string]interface{}类型，参数无法注入")
			return false
		}
	}

	retValues := customizeMatch.funValue.Call(in)
	if len(retValues) == 1 {
		if retValues[0].Bool() {
			customizeMatch.SetBlackMsg("属性 %v 的值 %v 命中禁用条件回调 [%v] ", field.Name, fieldValue, customizeMatch.expression)
		} else {
			customizeMatch.SetWhiteMsg("属性 %v 的值 %v 没命中只允许条件回调 [%v] ", field.Name, fieldValue, customizeMatch.expression)
		}
		return retValues[0].Bool()
	} else {
		kind0 := retValues[0].Kind()
		kind1 := retValues[1].Kind()

		if kind0 == reflect.Bool {
			if retValues[0].Bool() {
				customizeMatch.SetBlackMsg(retValues[1].String())
			} else {
				customizeMatch.SetWhiteMsg(retValues[1].String())
			}
			return retValues[0].Bool()
		} else if kind1 == reflect.Bool {
			if retValues[1].Bool() {
				customizeMatch.SetBlackMsg(retValues[0].String())
			} else {
				customizeMatch.SetWhiteMsg(retValues[0].String())
			}
			return retValues[1].Bool()
		} else {
			return retValues[0].Bool()
		}
	}
}

func (customizeMatch *CustomizeMatch) IsEmpty() bool {
	return customizeMatch.expression == ""
}

func BuildCustomizeMatcher(objectTypeFullName string, fieldKind reflect.Kind, objectFieldName string, tagName string, subCondition string, errMsg string) {
	if constant.MATCH != tagName {
		return
	}

	if !strings.Contains(subCondition, constant.Customize) {
		return
	}

	index := strings.Index(subCondition, "=")
	expression := subCondition[index+1:]

	if expression == "" {
		return
	}

	fun, contain := funMap[expression]
	if !contain {
		log.Warnf("the name of fun not find, funName is [%v]", expression)
		return
	}
	addMatcher(objectTypeFullName, objectFieldName, &CustomizeMatch{funValue: reflect.ValueOf(fun), expression: expression}, errMsg, true)
}

func RegisterCustomize(funName string, fun interface{}) {
	funValue := reflect.ValueOf(fun)
	if funValue.Kind() != reflect.Func {
		log.Warnf("fun is not fun[%v] type", funName)
		return
	}

	if funValue.Type().NumIn() > 3 {
		log.Warnf("the num of fun[%v] argument need to be less than or equal to 3", funName)
		return
	}

	if funValue.Type().NumOut() > 2 {
		log.Warnf("the num of fun[%v] return need to be less than or equal to 2", funName)
		return
	}

	if funValue.Type().NumOut() == 0 {
		log.Warnf("the type of fun[%v] return must be bool", funName)
		return
	} else if funValue.Type().NumOut() == 1 {
		if funValue.Type().Out(0).Kind() != reflect.Bool {
			log.Warnf("the type of fun[%v] return must be bool", funName)
			return
		}
	} else if funValue.Type().NumOut() == 2 {
		kind0 := funValue.Type().Out(0).Kind()
		kind1 := funValue.Type().Out(1).Kind()

		if kind0 != reflect.Bool && kind0 != reflect.String {
			log.Warnf("return type of fun[%v] return must be bool or string", funName)
			return
		}

		if kind1 != reflect.Bool && kind1 != reflect.String {
			log.Warnf("return type of fun[%v] return must be bool or string", funName)
			return
		}
	}

	funMap[funName] = fun
}
