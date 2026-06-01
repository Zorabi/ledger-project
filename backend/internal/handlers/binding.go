package handlers

import (
	"errors"
	"net/http"
	"reflect"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func bind(c *gin.Context, out any) bool {
	if err := c.ShouldBindJSON(out); err != nil {
		fail(c, http.StatusBadRequest, validationMessage(err, out))
		return false
	}
	return true
}

func validationMessage(err error, out any) string {
	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		return err.Error()
	}
	if len(validationErrors) == 0 {
		return "请求参数不合法"
	}
	field := jsonFieldName(out, validationErrors[0].StructNamespace())
	switch validationErrors[0].Tag() {
	case "required":
		return field + "为必填项"
	case "min":
		return field + "长度不能少于 " + validationErrors[0].Param() + " 位"
	default:
		return field + "格式不正确"
	}
}

func validatePassword(password string) string {
	if len([]rune(password)) < 10 {
		return "密码长度不能少于 10 位"
	}
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsSpace(r):
			return "密码不能包含空白字符"
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}
	missing := []string{}
	if !hasUpper {
		missing = append(missing, "大写字母")
	}
	if !hasLower {
		missing = append(missing, "小写字母")
	}
	if !hasDigit {
		missing = append(missing, "数字")
	}
	if !hasSpecial {
		missing = append(missing, "特殊字符")
	}
	if len(missing) > 0 {
		return "密码必须包含" + strings.Join(missing, "、")
	}
	return ""
}

func jsonFieldName(out any, namespace string) string {
	parts := strings.Split(namespace, ".")
	t := reflect.TypeOf(out)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	name := "字段"
	for _, part := range parts[1:] {
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			break
		}
		field, ok := t.FieldByName(part)
		if !ok {
			return part
		}
		fieldName := strings.Split(field.Tag.Get("json"), ",")[0]
		if fieldName != "" && fieldName != "-" {
			name = fieldName
		} else {
			name = part
		}
		t = field.Type
	}
	return name
}
