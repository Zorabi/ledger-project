package utils

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zhTranslations "github.com/go-playground/validator/v10/translations/zh"
)

var trans ut.Translator

func InitTrans(locale string) (err error) {
	// 修改gin框架中的Validator引擎属性
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		// 注册获取json tag的自定义方法
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})
		zhT := zh.New() // 中文翻译器
		uni := ut.New(zhT, zhT)

		var ok bool
		trans, ok = uni.GetTranslator(locale)
		if !ok {
			return fmt.Errorf("translator %s not found", locale)
		}

		err = zhTranslations.RegisterDefaultTranslations(v, trans)
	}
	return
}

func Translate(err error) map[string][]string {
	var result = make(map[string][]string)
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		for _, err := range validationErrors {
			result[err.Field()] = append(result[err.Field()], err.Translate(trans))
		}
	}
	return result
}
