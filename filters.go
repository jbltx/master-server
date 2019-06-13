package main

import (
	"strings"
)

type filter struct {
	key   string
	value interface{}
}

func isValidFilterKey(key string) bool {

	switch key {
	case "nor":
	case "nand":
	case "dedicated":
	case "secure":
	case "gamedir":
	case "map":
	case "linux":
	case "password":
	case "empty":
	case "full":
	case "proxy":
	case "appid":
	case "napp":
	case "noplayers":
	case "white":
	case "gametype":
	case "gamedata":
	case "gamedataor":
	case "name_match":
	case "version_match":
	case "collapse_addr_hash":
	case "gameaddr":
		return true
	default:
		break
	}

	return false
}

func newFilter(filterStr string) *filter {

	split := strings.Split(filterStr, "\\")

	if len(split) == 3 {
		key := split[1]
		value := split[2]

		if isValidFilterKey(key) {
			return &filter{
				key:   key,
				value: value,
			}
		}
	}

	return nil
}
