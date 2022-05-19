package main

import (
	"fmt"
	"reflect"
	"strings"

	"bitbucket.org/wimarksystems/wmcli/internal"

	"github.com/google/uuid"
)

type Location struct {
	ID   string `json:"id" bson:"_id"`
	Name string `json:"name" bson:"name"`
}

type LocationObject struct {
	Model   string `json:"model" bson:"model"`
	ModelID string `json:"model_id" bson:"model_id"`
}

type Doc map[string]interface{}

func stringsJoinQuotes(in []string) []string {
	ret := []string{}
	b := ""
	quotes := false
	inLength := len(in)
	for i, j := range in {
		if quotes {
			b += " " + j
			if strings.HasSuffix(j, "\"") || i == inLength-1 {
				quotes = false
				ret = append(ret, strings.TrimSpace(b))
			}
		} else {
			b = j
			if strings.Contains(strings.TrimSuffix(j, "\""), "\"") {
				if strings.HasSuffix(j, "\"") || i == inLength-1 {
					ret = append(ret, b)
				} else {
					quotes = true
				}
			} else {
				ret = append(ret, b)
			}
		}
	}
	return ret
}

func hasbit(a, b uint8) bool {
	return a&b == b
}

func isValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

func newUUID() string {
	return uuid.New().String()
}

func contains(s []string, e string) string {
	for _, a := range s {
		if a == e {
			return a
		}
	}
	return ""
}

func stringsContainsExMapKey(s []string, e string) string {
	for _, a := range s {
		if strings.Contains(a, "*") {
			asplit := strings.Split(a, ".")
			esplit := strings.Split(e, ".")
			if len(asplit) == len(esplit) {
				res := true
				for i := range asplit {
					if asplit[i] != esplit[i] {
						if asplit[i] != "*" {
							res = false
							break
						}
					}
				}
				if res {
					return a
				}
			}
		}
	}
	return ""
}

func mapKeysToList(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func printResult(in map[string]interface{}, prefix, opt string) {
	mlen := 0
	for k := range in {
		if len(k) > mlen {
			mlen = len(k)
		}
	}
	for k, v := range in {
		if k == "_id" {
			continue
		}
		if cfg.UseDbOutput {
			kstruct, ok := internal.BsonToStructMap[opt][k]
			if ok {
				k = kstruct
			}
		}
		if reflect.TypeOf(v).Kind() == reflect.Map {
			printResult(v.(map[string]interface{}), prefix+k+".", opt)
		} else {
			fmt.Printf("%s%s=%s\n", prefix, k, v)
		}
	}
}

func printResultPretty(in map[string]interface{}, prefix, opt string) {
	mlen := 0
	for k := range in {
		kstruct, ok := internal.BsonToStructMap[opt][k]
		if ok {
			k = kstruct
		}
		if len(k) > mlen {
			mlen = len(k)
		}
	}
	for k, v := range in {
		if k == "_id" {
			continue
		}
		kstruct, ok := internal.BsonToStructMap[opt][k]
		if ok {
			k = kstruct
		}
		clen := len(k)
		if reflect.TypeOf(v).Kind() == reflect.Map {
			fmt.Printf("%s%s: \n", prefix, k)
			printResultPretty(v.(map[string]interface{}), prefix+"  ", opt)
		} else {
			if reflect.ValueOf(v).IsZero() {
				v = "<nil " + fmt.Sprint(reflect.TypeOf(v)) + ">"
			}
			pad := strings.Repeat(".", mlen-clen+5)
			fmt.Printf("%s%s:%s%s\n", prefix, k, pad, v)
		}
	}
}
