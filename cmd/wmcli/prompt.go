package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	prompt "github.com/c-bata/go-prompt"

	"bitbucket.org/wimarksystems/wmcli/internal"
)

const (
	tOperation uint8 = 1 << iota
	tObjectID
	tOption
	tField
	tError
)

var promptIsInteractive bool = true

type OptionPair struct {
	Field string
	Value string
}

type Command struct {
	Operation string
	Options   map[string][]OptionPair
	ObjectIDs map[string][]string

	LastOption string
	LastField  string
	LastValue  string
}

func suggestUUID(opt string) []prompt.Suggest {
	return []prompt.Suggest{
		{
			Text:        newUUID(),
			Description: "new uuid",
		},
	}
}

func suggestOperation() []prompt.Suggest {
	s := []prompt.Suggest{}
	for k, v := range internal.ModelOperations {
		s = append(s, prompt.Suggest{
			Text:        k,
			Description: v,
		})
	}
	return s
}

func suggestOption() []prompt.Suggest {
	s := []prompt.Suggest{}
	for k, v := range internal.ModelOptions {
		s = append(s, prompt.Suggest{
			Text:        k,
			Description: v,
		})
	}
	return s
}

func suggestField(opt string) []prompt.Suggest {
	s := []prompt.Suggest{}
	if cmd.Operation == "set" {
		mfields := internal.ModelFields
		for k, v := range mfields[opt] {
			s = append(s, prompt.Suggest{
				Text:        k + "=",
				Description: v,
			})
		}
	} else {
		for k, v := range internal.ModelFields[opt] {
			s = append(s, prompt.Suggest{
				Text:        k,
				Description: v,
			})
		}
	}
	return s
}

func promptIsValidPair(in, opt string, isSet bool) bool {
	if !isSet {
		if contains(mapKeysToList(internal.ModelFields[opt]), in) != "" {
			return true
		}
		if stringsContainsExMapKey(mapKeysToList(internal.ModelFields[opt]), in) != "" {
			return true
		}
		return false
	}
	split := strings.Split(in, "=")
	if len(split) != 2 {
		return false
	}
	field := ""
	mfields := internal.ModelFields
	field = contains(mapKeysToList(mfields[opt]), split[0])
	if field == "" {
		field = stringsContainsExMapKey(mapKeysToList(mfields[opt]), split[0])
	}
	if field == "" {
		return false
	}
	if strings.Contains(field, "[RO]") {
		return false
	}
	atype := mfields[opt][field]
	switch atype {
	case "string", "*string":
		if promptIsInteractive {
			if strings.HasPrefix(split[1], "\"") && strings.HasSuffix(split[1], "\"") {
				return true
			}
		} else {
			return true
		}
	case "int":
		if _, err := strconv.Atoi(split[1]); err == nil {
			return true
		}
	case "bool":
		if contains([]string{"false", "true"}, split[1]) != "" {
			return true
		}
	}
	return false
}

func promptVerifier(args []string) []uint8 {
	cmd = Command{}
	abits := make([]uint8, len(args))
	cmd.Options = make(map[string][]OptionPair)
	cmd.ObjectIDs = make(map[string][]string)
	for i, arg := range args {
		if i == 0 {
			abits[i] += tOperation
		} else {
			abits[i] += tOption
			if cmd.LastOption != "" {
				if hasbit(abits[i-1], tOption) {
					abits[i] += tObjectID
				}
				abits[i] += tField
			}
			switch cmd.Operation {
			case "set":
				if hasbit(abits[i-1], tOption) {
					abits[i] -= tOption
				}
			case "delete":
				if hasbit(abits[i], tObjectID) {
					abits[i] = tObjectID
				}
			}
		}
		if hasbit(abits[i], tOperation) {
			if contains(mapKeysToList(internal.ModelOperations), arg) == "" {
				abits[i] += tError
			} else {
				cmd.Operation = arg
				continue
			}
		}
		if hasbit(abits[i], tObjectID) {
			if !isValidUUID(arg) {
				if !hasbit(abits[i], tError) {
					abits[i] += tError
				}
			} else {
				if hasbit(abits[i], tError) {
					abits[i] -= tError
				}
				cmd.ObjectIDs[cmd.LastOption] = append(cmd.ObjectIDs[cmd.LastOption], arg)
				continue
			}
		}
		if hasbit(abits[i], tOption) {
			if contains(mapKeysToList(internal.ModelOptions), arg) == "" {
				if !hasbit(abits[i], tError) {
					abits[i] += tError
				}
			} else {
				cmd.LastOption = arg
				cmd.Options[arg] = append(cmd.Options[arg], OptionPair{
					Field: "",
					Value: "",
				})
				if hasbit(abits[i], tError) {
					abits[i] -= tError
				}
				continue
			}
		}
		if hasbit(abits[i], tField) {
			if !promptIsValidPair(arg, cmd.LastOption, cmd.Operation == "set") {
				if !hasbit(abits[i], tError) {
					abits[i] += tError
				}
			} else {
				s := strings.Split(arg, "=")
				if len(s) == 1 {
					s = append(s, "")
				}
				cmd.Options[cmd.LastOption] = append(cmd.Options[cmd.LastOption], OptionPair{
					Field: s[0],
					Value: s[1],
				})
				if hasbit(abits[i], tError) {
					abits[i] -= tError
				}
				continue
			}
		}
	}
	return abits
}

func promptCompleter(in prompt.Document) []prompt.Suggest {
	text := in.TextBeforeCursor()
	s := []prompt.Suggest{}
	testFields := strings.Fields(text)
	lchar, _ := utf8.DecodeLastRuneInString(text)
	if unicode.IsSpace(lchar) || len(testFields) == 0 {
		testFields = append(testFields, " ")
	}
	text_final := stringsJoinQuotes(testFields)
	// TODO fix strange bug with promptVerifier and spaces in string set value
	r := promptVerifier(text_final)
	hasError := false
	for i, v := range r {
		if hasbit(v, tError) {
			if i != len(r)-1 {
				hasError = true
			}
			break
		}
	}
	if hasError {
		return s
	}
	lr := r[len(r)-1]
	if hasbit(lr, tOperation) {
		s = append(s, suggestOperation()...)
	}
	if hasbit(lr, tObjectID) {
		s = append(s, suggestUUID(cmd.LastOption)...)
	}
	if hasbit(lr, tOption) {
		s = append(s, suggestOption()...)
	}
	if hasbit(lr, tField) {
		s = append(s, suggestField(cmd.LastOption)...)
	}
	return prompt.FilterFuzzy(s, in.GetWordBeforeCursor(), true)
}

func promptExec(in string) {
	if in == "exit" || in == "quit" {
		handleExit()
		os.Exit(0)
	}
	text_final := stringsJoinQuotes(strings.Fields(strings.TrimSpace(in)))
	r := promptVerifier(text_final)
	hasError := false
	for _, v := range r {
		if hasbit(v, tError) {
			hasError = true
			break
		}
	}

	if hasError {
		fmt.Println("Error in command!")
		return
	} else {
		doCommand(cmd)
	}
}

func promptExecNoninteractive(in []string) {
	if in[0] == "exit" || in[0] == "quit" {
		handleExit()
		os.Exit(0)
	}
	r := promptVerifier(in)
	hasError := false
	for _, v := range r {
		if hasbit(v, tError) {
			hasError = true
			break
		}
	}

	if hasError {
		fmt.Println("Error in command!")
		return
	} else {
		doCommand(cmd)
	}
}
