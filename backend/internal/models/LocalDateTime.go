package models

import (
	"database/sql/driver"
	"reflect"
	"time"
)

type LocalDateTime time.Time

const TimeFormat = "2006-01-02 15:04:05"

func (localDateTime LocalDateTime) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len(TimeFormat)+2)
	b = append(b, '"')
	b = time.Time(localDateTime).AppendFormat(b, TimeFormat)
	b = append(b, '"')
	return b, nil
}

func (localDateTime *LocalDateTime) UnmarshalJSON(data []byte) (err error) {
	if len(data) == 2 {
		*localDateTime = LocalDateTime(time.Time{})
		return
	}

	now, err := time.Parse(`"`+TimeFormat+`"`, string(data))
	*localDateTime = LocalDateTime(now)
	return
}

func (localDateTime LocalDateTime) Value() (driver.Value, error) {
	if localDateTime.String() == "0001-01-01 00:00:00" {
		return nil, nil
	}
	return []byte(time.Time(localDateTime).Format(TimeFormat)), nil
}

func (localDateTime *LocalDateTime) Scan(v interface{}) error {
	tTime, _ := time.Parse("2006-01-02 15:04:05 +0800 CST", v.(time.Time).String())
	*localDateTime = LocalDateTime(tTime)
	return nil
}

func (localDateTime LocalDateTime) String() string {
	return time.Time(localDateTime).Format(TimeFormat)
}

func ValidateJSONDateType(field reflect.Value) interface{} {
	if field.Type() == reflect.TypeOf(LocalDateTime{}) {
		timeStr := field.Interface().(LocalDateTime).String()
		// 0001-01-01 00:00:00 是 go 中 time.Time 类型的空值
		// 这里返回 Nil 则会被 validator 判定为空值，而无法通过 `binding:"required"` 规则
		if timeStr == "0001-01-01 00:00:00" {
			return nil
		}
		return timeStr
	}
	return nil
}
