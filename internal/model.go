package internal

import (
	"reflect"
	"strings"

	wimark "github.com/wimark/libwimark"
)

var ModelOperations = map[string]string{
	"read":   "Read",
	"set":    "Set",
	"delete": "Delete",
}

var ModelOptions = map[string]string{
	"wlan":         "WLAN",
	"radius":       "RADIUS",
	"cpe":          "CPE",
	"cpe_template": "CPE Template",
	"cpe_model":    "CPE Model",
}

var ModelFields = initModelFields()
var ModelFieldsDB = map[string]map[string]string{}
var ModelFieldsExceptions = []string{
	"wlanSecurity",
	"wlanRadiusAcctInterval",
	"wlanRadiusAcctMirroring",
	"wlanFilterMode",
	"wlanWhiteList.#",
	"wlanBlackList.#",
	"wlanFirewall",
	"wlanWMMConfig.Categories.*",
	"wlanNATNetwork",
	"wlanGuestControl",
	"wlanBeelineAccountingType",
	"wlanSignalConnect",
	"wlanSignalStay",
	"wlanSignalStrikes",
	"wlanSignalPollTime",
	"wlanSignalDropReason",
	"wlanNASGenerate",
	"cpeConnected",
	"cpeModel.Id",
	"cpeConfigStatus",
	"cpeLastError",
	"cpeConfig.Wired.*",
	"cpeConfig.LbsConfig.Enabled",
	"cpeConfig.LogConfig.LogIP",
	"cpeConfig.DHCPCapConfig.Enabled",
	"cpeConfig.Firewall",
	"cpeConfig.Firmware.FileUrl",
	"cpeConfig.Beeline.NASIP",
	"cpeConfig.WiFiLock",
	"cpeConfig.NetManual.*",
	"cpeConfig.WifiManual.*",
	"cpeState",
	"cpeFirstConnection",
	"cpeLastConnection",
	"cpeLastDisconnection",
	"cpeConfigNotSend",
	"cpe_templateTemplate.CpeConfig.Wired.*",
	"cpe_templateTemplate.CpeConfig.LbsConfig.Enabled",
	"cpe_templateTemplate.CpeConfig.LogConfig.LogIP",
	"cpe_templateTemplate.CpeConfig.DHCPCapConfig.Enabled",
	"cpe_templateTemplate.CpeConfig.Firewall",
	"cpe_templateTemplate.CpeConfig.Firmware.StorageUrl",
	"cpe_templateTemplate.CpeConfig.Beeline.NASIP",
	"cpe_templateTemplate.CpeConfig.WiFiLock",
	"cpe_templateTemplate.CpeConfig.NetManual.*",
	"cpe_templateTemplate.CpeConfig.WifiManual.*",
	"cpe_modelName",
}

var ModelQtechInternals = map[string]string{
	"QWP-320-AC-VC":       "xd3200",
	"QWO-320-AC-VC":       "xd3200",
	"QWO-95-AC-VC":        "xd3200",
	"QWP-65-AC-VC":        "yuncore,xd6800",
	"QW0-65-AC-VC":        "yuncore,xd6800",
	"QWP-420-AC-VC-Alpha": "yuncore,xd4200",
	"QWO-420-AC-VC-Alpha": "yuncore,xd4200",
	"QWP-420-AC-VC-Beta":  "inspur,iap5820i-l",
	"QWO-420-AC-VC-Beta":  "inspur,iap5820i-l",
}

var ModelFrequencyRadioName = initModelQtechFreqRadioName()

var BsonToStructMap map[string]map[string]string = make(map[string]map[string]string)
var StructToJsonMap map[string]map[string]string = make(map[string]map[string]string)

func addmaps(field reflect.StructField, opt string) {
	bsonv := field.Tag.Get("bson")
	jsonv := field.Tag.Get("json")
	structv := field.Name

	if bsonv == ",inline" {
		bsonv = ""
	}
	if bsonv == "" {
		bsonv = jsonv
	}

	bsonv = strings.ReplaceAll(bsonv, ",inline", "")
	bsonv = strings.ReplaceAll(bsonv, ",omitempty", "")
	jsonv = strings.ReplaceAll(jsonv, ",inline", "")
	jsonv = strings.ReplaceAll(jsonv, ",omitempty", "")

	if bsonv != "" {
		_, ok := BsonToStructMap[opt]
		if ok {
			BsonToStructMap[opt][bsonv] = structv
		} else {
			BsonToStructMap[opt] = map[string]string{
				bsonv: structv,
			}
		}
	}
	if jsonv != "" {
		_, ok := StructToJsonMap[opt]
		if ok {
			StructToJsonMap[opt][structv] = jsonv
		} else {
			StructToJsonMap[opt] = map[string]string{
				structv: jsonv,
			}
		}
	}
}

func fieldig(in reflect.Type, out *map[string]map[string]string, opt, prefix string) {
	for _, field := range reflect.VisibleFields(in) {
		addmaps(field, opt)
		kfmt := ""
		if opt == "cpe_template" {
			tagfield := field.Tag.Get("bson")
			if tagfield == "" {
				tagfield = field.Tag.Get("json")
			}
			if tagfield == "" {
				tagfield = strings.ToLower(field.Name)
			}

			if prefix == "" {
				kfmt = tagfield
			} else {
				kfmt = prefix + "." + tagfield
			}
		} else {
			if prefix == "" {
				kfmt = field.Name
			} else {
				kfmt = prefix + "." + field.Name
			}
		}
		t := field.Type
		tstr := field.Type.Kind().String()
		switch t.Kind() {
		case reflect.Slice:
			kfmt += ".#"
			t = field.Type.Elem()
			tstr = t.Kind().String()
		case reflect.Ptr:
			t = field.Type.Elem()
			tstr = "*" + t.Kind().String()
		case reflect.Map:
			kfmt += ".*"
			t = field.Type.Elem()
			tstr = t.Kind().String()
		}

		exception := false
		for _, ex := range ModelFieldsExceptions {
			if opt+kfmt == ex {
				exception = true
				break
			}
		}
		if t.Kind() == reflect.Interface {
			exception = true
		}

		if exception {
			kfmt += "[RO]"
		}

		if t.Kind() == reflect.Struct {
			fieldig(t, out, opt, kfmt)
		} else {
			(*out)[opt][kfmt] = tstr
		}

	}
}

func initModelFields() map[string]map[string]string {
	fields := make(map[string]map[string]string)
	for i := range ModelOptions {
		fields[i] = make(map[string]string)
	}

	// CPE_TEMPLATE_FIELD
	type FromTemplate struct {
		FromTemplate wimark.UUID `json:"from_template"`
	}

	// EXCEPTIONS
	fieldig(reflect.TypeOf(wimark.WPA2EnterpriseData{}), &fields, "wlan", "Security.wpa2enterprise")
	fieldig(reflect.TypeOf(wimark.WPA2PersonalData{}), &fields, "wlan", "Security.wpa2personal")
	fieldig(reflect.TypeOf(wimark.WPAEnterpriseData{}), &fields, "wlan", "Security.wpaenterprise")
	fieldig(reflect.TypeOf(wimark.WPAPersonalData{}), &fields, "wlan", "Security.wpapersonal")
	fieldig(reflect.TypeOf(FromTemplate{}), &fields, "cpe", "")

	// MAIN
	fieldig(reflect.TypeOf(wimark.WLAN{}), &fields, "wlan", "")
	fieldig(reflect.TypeOf(wimark.Radius{}), &fields, "radius", "")
	fieldig(reflect.TypeOf(wimark.CPE{}), &fields, "cpe", "")
	fieldig(reflect.TypeOf(wimark.ConfigRule{}), &fields, "cpe_template", "")
	fieldig(reflect.TypeOf(wimark.CPEModel{}), &fields, "cpe_model", "")

	return fields
}

func models() []string {
	var l = make([]string, len(ModelQtechInternals))
	var i = 0
	for k := range ModelQtechInternals {
		l[i] = k
		i++
	}
	return l
}

func modelBands(a map[int][]string) []int {
	var l = make([]int, len(a))
	var i = 0
	for k := range a {
		l[i] = k
		i++
	}
	return l
}

func initModelQtechFreqRadioName() map[string]map[int][]string {
	var a = map[string]map[int][]string{}
	a["xd3200"] = map[int][]string{
		2: {"radio1"},
		5: {"radio0"},
	}
	a["yuncore,xd4200"] = map[int][]string{
		2: {"radio1"},
		5: {"radio0"},
	}
	a["yuncore,xd6800"] = map[int][]string{
		2: {"radio1"},
		5: {"radio0", "radio2"},
	}
	return a
}
