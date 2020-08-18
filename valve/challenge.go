package valve

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
)

const challengeTagName string = "challenge"

type ChallengeRequest struct {
	Protocol       int32           `challenge:"protocol"`
	ChallengeValue int32           `challenge:"challenge"`
	Players        int32           `challenge:"players"`
	Max            int32           `challenge:"max"`
	Bots           bool            `challenge:"bots"`
	GameDir        string          `challenge:"gamedir"`
	Map            string          `challenge:"map"`
	Password       bool            `challenge:"password"`
	OS             OperatingSystem `challenge:"os"`
	Lan            bool            `challenge:"lan"`
	Region         Region          `challenge:"region"`
	Type           string          `challenge:"type"`
	Secure         bool            `challenge:"secure"`
	Version        string          `challenge:"version"`
	Product        string          `challenge:"product"`
}

func UnmarshallChallenge(message []byte, ret interface{}) error {
	msg := string(message)
	msg = strings.Trim(msg, "\n")
	v := reflect.ValueOf(ret).Elem()
	t := v.Type()
	for fi := 0; fi < v.NumField(); fi++ {
		field := v.Field(fi)
		tag := t.Field(fi).Tag.Get(challengeTagName)
		if len(tag) > 0 {
			tags := strings.Split(tag, ",")
			fieldName := "\\" + tags[0] + "\\"
			startField := strings.Index(msg, fieldName)
			if startField >= 0 {
				startValue := startField + len(fieldName)
				endValue := strings.Index(msg[startValue:], "\\") + startValue
				var valStr string
				if endValue > startValue {
					valStr = msg[startValue:endValue]
				} else {
					valStr = msg[startValue:]
				}
				switch field.Kind() {
				case reflect.Bool:
					field.SetBool(valStr == "1")
				case reflect.Int8:
				case reflect.Int16:
				case reflect.Int32:
					val, err := strconv.Atoi(valStr)
					if err != nil {
						return errors.New("The field " + fieldName + " value can't be parsed as an integer")
					}
					field.SetInt(int64(val))
				case reflect.String:
					field.SetString(valStr)
				case reflect.Uint8:
					val, err := strconv.Atoi(valStr)
					if err != nil {
						return errors.New("The field " + fieldName + " value can't be parsed as an unsigned integer")
					}
					field.SetUint(uint64(val))
				default:
					return errors.New("The field " + fieldName + " value has an unsupported type (" + field.Kind().String() + ")")
				}

			} else {
				return errors.New("The field " + fieldName + " hasn't been found in the message")
			}
		}
	}
	return nil
}
