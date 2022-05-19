package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/wimarksystems/wmcli/internal"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	wimark "github.com/wimark/libwimark"

	"github.com/globalsign/mgo/bson"
	"github.com/imdario/mergo"
	"github.com/rs/xid"
)

var rootLocationID string

func fielddig(in map[string]interface{}, fields []string) map[string]interface{} {
	ret := map[string]interface{}{}
	for k, v := range in {
		if len(fields) > 0 {
			if contains(fields, k) == "" {
				continue
			}
		}
		if v == nil {
			ret[k] = "<nil pointer>"
			continue
		}
		switch reflect.TypeOf(v).Kind() {
		case reflect.Map:
			ret[k] = fielddig(v.(bson.M), fields)

		case reflect.Slice:
			if len(v.([]interface{})) > 0 {
				for i, elem := range v.([]interface{}) {
					elemmap, ok := elem.(bson.M)
					if ok {
						id, ok := elemmap["_id"]
						if !ok {
							id = strconv.Itoa(i)
						}
						ret[k+".["+id.(string)+"]"] = fielddig(elemmap, fields)
					} else {
						ret[k+".["+strconv.Itoa(i)+"]"] = fmt.Sprint(reflect.ValueOf(v))
					}
				}
			} else {
				ret[k] = fmt.Sprint(reflect.ValueOf(v))
			}

		default:
			if reflect.ValueOf(v).IsZero() {
				ret[k] = "<nil " + fmt.Sprint(reflect.TypeOf(v).Kind()) + ">"
			} else {
				ret[k] = fmt.Sprint(reflect.ValueOf(v))
			}
		}
	}
	return ret
}

func getLocationByName(name string) *Location {
	tmp := []Location{}
	queryMap := Doc{
		"name": name,
	}
	err := db.Find("location", queryMap, &tmp)
	if err != nil {
		return nil
	}
	if len(tmp) == 0 {
		return nil
	}
	return &tmp[0]
}

func hasModelInLocation(id, model_id string) bool {
	tmp := []Location{}
	queryMap := Doc{
		"_id":            id,
		"items.model_id": model_id,
	}
	err := db.Find("location", queryMap, &tmp)
	if err != nil {
		fmt.Println("[location check]" + err.Error())
		return false
	}
	if len(tmp) == 0 {
		return false
	}
	return true
}

func addToRootLocation(model, model_id string) {

	find := Doc{
		"_id": model + model_id,
	}
	set := Doc{
		"location_id": rootLocationID,
		"model":       model,
		"model_id":    model_id,
	}
	err := db.UpsertWithQuery("base_location", find, set)
	if err != nil {
		fmt.Println("[base location] " + err.Error())
		return
	}

	if !hasModelInLocation(rootLocationID, model_id) {

		q := Doc{
			"_id": rootLocationID,
		}

		s := Doc{
			"$push": Doc{
				"items": LocationObject{
					Model:   model,
					ModelID: model_id,
				},
			},
		}
		err := db.UpdateWithQuery("location", q, s)
		if err != nil {
			fmt.Println("[location] " + err.Error())
		}

	}
}

func dbRead(coll string, ids, fields []string) []bson.M {
	for i, v := range fields {
		v = strings.ReplaceAll(v, "[RO]", "")
		vb, ok := internal.StructToJsonMap[coll][v]
		if ok {
			fields[i] = vb
		}
	}
	switch coll {
	case "wlan":
		coll = "wlans"
	case "cpe":
		coll = "cpes"
	case "radius":
		coll = "radius"
	case "cpe_template":
		coll = "config_rule"
	case "cpe_model":
		coll = "cpe_model"
	default:
		return []bson.M{}
	}
	result := []bson.M{}
	query := []bson.M{}
	if len(ids) > 0 {
		query = []bson.M{
			{
				"$match": bson.M{
					"_id": bson.M{
						"$in": ids,
					},
				},
			},
		}
	}
	err := db.Pipe(coll, query, &result)
	if err != nil {
		fmt.Println(err)
		return []bson.M{}
	}

	if len(fields) > 0 {
		fmt.Println("Filtering by fields:")
		fmt.Println(fields)
	}

	return result
}

func sendMQTTEvent(client mqtt.Client, eventType wimark.SystemEventType,
	description string,
	data interface{}) error {
	mod := wimark.ModuleCLI

	t := wimark.EventTopic{
		SenderModule: mod,
		SenderID:     "",
		Type:         eventType,
	}

	eventPayload := wimark.SystemEvent{
		Subject_id:  mod.String(),
		Timestamp:   time.Now().Unix(),
		Level:       wimark.SystemEventLevelINFO,
		Description: description,
	}
	eventObject := wimark.SystemEventObject{
		Type: eventType,
		Data: data,
	}
	eventPayload.SystemEventObject = eventObject
	payload, _ := json.Marshal(eventPayload)

	if token := client.Publish(t.TopicPath(), 2,
		false, payload); token.Wait() && token.Error() != nil {
		err := token.Error()
		return err
	}
	return nil
}

func doCPECondigRule(op wimark.Operation, oid string, model interface{}) error {

	model.(map[string]interface{})["_id"] = oid

	switch op {
	case wimark.OperationCreate:
		b, err := bson.Marshal(wimark.ConfigRule{})
		if err != nil {
			return err
		}
		rsp := map[string]interface{}{}
		err = bson.Unmarshal(b, &rsp)
		if err != nil {
			return err
		}

		model.(map[string]interface{})["is_auto"] = true

		mergo.Merge(&rsp, model)
		db.Insert("config_rule", rsp)

	case wimark.OperationUpdate:
		db.Update("config_rule", oid, model)

	case wimark.OperationDelete:
		db.Remove("config_rule", oid)
	}

	return nil
}

func doMQTTRequest(op wimark.Operation, opt, oid string, options []OptionPair) error {

	dbopt := opt
	switch opt {
	case "wlan":
		opt = "wlan"
	case "cpe":
		opt = "cpe"
	case "radius":
		opt = "radius"
	case "cpe_template":
		opt = "cpe_config_templates"
	case "cpe_model":
		return fmt.Errorf("WIP ability to CUD cpe models")
	default:
		return fmt.Errorf("wrong model passed")
	}

	set_empty := false
	db_results := dbRead(dbopt, []string{oid}, []string{})

	if len(db_results) == 0 {
		switch op {
		case wimark.OperationUpdate:
			op = wimark.OperationCreate
			opt_store := opt
			if opt_store[len(opt_store)-1] != 's' {
				opt_store += "s"
			}
			addToRootLocation(opt_store, oid)
		case wimark.OperationDelete:
			return fmt.Errorf("attempt to delete nonexistent object id")
		}
	}

	if op == wimark.OperationDelete && len(options) > 1 {
		return fmt.Errorf("WIP ability to delete (set null) exact fields, please use set <nil of value type>")
		// TODO WIP
		set_empty = true
		op = wimark.OperationUpdate
	}

	var payload map[string]interface{}

	var modelObject interface{} = map[string]interface{}{}
	if op != wimark.OperationDelete {
		for _, v := range options {
			cursor := modelObject
			var pcursor interface{}
			var pjsonv string
			vsplit := strings.Split(v.Field, ".")
			for i, j := range vsplit {
				if j == "" {
					continue
				}
				jsonv := internal.StructToJsonMap[opt][j]
				if jsonv == "" {
					jsonv = j
				}
				if jsonv == "#" {
					if i == len(vsplit)-1 {
						if set_empty {
							pcursor.(map[string]interface{})[pjsonv] = []interface{}{}
						} else {
							if reflect.TypeOf(v.Value).Kind() == reflect.String {
								switch v.Value {
								case "true":
									pcursor.(map[string]interface{})[pjsonv] = append(pcursor.(map[string]interface{})[pjsonv].([]interface{}), true)
								case "false":
									pcursor.(map[string]interface{})[pjsonv] = append(pcursor.(map[string]interface{})[pjsonv].([]interface{}), false)
								default:
									val := v.Value
									if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") {
										val = strings.TrimSuffix(strings.TrimPrefix(val, "\""), "\"")
										pcursor.(map[string]interface{})[pjsonv] = append(pcursor.(map[string]interface{})[pjsonv].([]interface{}), val)
									} else {
										ival, err := strconv.Atoi(val)
										if err != nil {
											pcursor.(map[string]interface{})[pjsonv] = append(pcursor.(map[string]interface{})[pjsonv].([]interface{}), v.Value)
										} else {
											pcursor.(map[string]interface{})[pjsonv] = append(pcursor.(map[string]interface{})[pjsonv].([]interface{}), ival)
										}
									}
								}
							} else {
								pcursor.(map[string]interface{})[pjsonv] = append(pcursor.(map[string]interface{})[pjsonv].([]interface{}), v.Value)
							}
						}
					} else {
						newbranch := map[string]interface{}{}
						pcursor.(map[string]interface{})[pjsonv] = append(pcursor.(map[string]interface{})[pjsonv].([]interface{}), newbranch)
						cursor = newbranch
					}
				} else {
					if _, ok := cursor.(map[string]interface{})[jsonv]; !ok {
						if i < len(vsplit)-1 && vsplit[i+1] == "#" {
							cursor.(map[string]interface{})[jsonv] = []interface{}{}

						} else {
							cursor.(map[string]interface{})[jsonv] = map[string]interface{}{}
						}
					}
					if i == len(vsplit)-1 {
						if set_empty {
							cursor.(map[string]interface{})[jsonv] = nil
						} else {
							if reflect.TypeOf(v.Value).Kind() == reflect.String {
								switch v.Value {
								case "true":
									cursor.(map[string]interface{})[jsonv] = true
								case "false":
									cursor.(map[string]interface{})[jsonv] = false
								default:
									val := v.Value
									if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") {
										val = strings.TrimSuffix(strings.TrimPrefix(val, "\""), "\"")
										cursor.(map[string]interface{})[jsonv] = val
									} else {
										ival, err := strconv.Atoi(val)
										if err != nil {
											cursor.(map[string]interface{})[jsonv] = v.Value
										} else {
											cursor.(map[string]interface{})[jsonv] = ival
										}
									}
								}
							} else {
								cursor.(map[string]interface{})[jsonv] = v.Value
							}
						}
					} else {
						pcursor = cursor
						pjsonv = jsonv
						cursor = cursor.(map[string]interface{})[jsonv]
					}
				}
			}
		}
		// Process wlan security exceptions
		security_data, ok := modelObject.(map[string]interface{})["security"]
		if ok {
			for k, v := range security_data.(map[string]interface{}) {
				modelObject.(map[string]interface{})["security"].(map[string]interface{})["type"] = k
				modelObject.(map[string]interface{})["security"].(map[string]interface{})["data"] = v
				delete(modelObject.(map[string]interface{})["security"].(map[string]interface{}), k)
				break
			}
		}
		if opt == "cpe" {
			tid, ok := modelObject.(map[string]interface{})["from_template"]
			if ok {
				if isValidUUID(tid.(string)) {
					db_templates := dbRead("cpe_template", []string{tid.(string)}, []string{})
					if len(db_templates) > 0 {
						db_template := db_templates[0]["template"].(bson.M)["cpe_config_template"]
						delete(modelObject.(map[string]interface{}), "from_template")
						wmodel := wimark.CPEConfig{}
						b, _ := bson.Marshal(db_template)
						bson.Unmarshal(b, &wmodel)
						for k, v := range db_template.(bson.M)["wifi"].(bson.M) {
							v.(bson.M)["_id"] = k
						}
						b, _ = bson.Marshal(db_template.(bson.M)["wifi"].(bson.M))
						wmodel.Wifi.SetBSON(bson.Raw{
							Kind: 0x04,
							Data: b,
						})
						// TODO FIXTHIS
						//b, _ = bson.Marshal(db_template.(bson.M)["wired"].(bson.M))
						//wmodel.Wired.SetBSON(bson.Raw{
						//	Kind: 0x04,
						//	Data: b,
						//})
						_, ok := db_template.(bson.M)["lbs_config"].(bson.M)["filter_mode"].(bson.M)
						if ok {
							b, _ = bson.Marshal(db_template.(bson.M)["lbs_config"].(bson.M)["filter_mode"].(bson.M))
							wmodel.LbsConfig.FilterMode.SetBSON(bson.Raw{
								Kind: 0x04,
								Data: b,
							})
						}
						j, _ := json.Marshal(wmodel)
						jmap := map[string]interface{}{}
						json.Unmarshal(j, &jmap)
						mergo.Merge(&jmap, modelObject, mergo.WithOverride)
						modelObject.(map[string]interface{})["config"] = jmap
						modelObject.(map[string]interface{})["name"] = wmodel.Name
						modelObject.(map[string]interface{})["description"] = wmodel.Description
					} else {
						fmt.Println("Failed to load template (no objects with such uuid)", tid)
					}
				} else {
					fmt.Println("Failed to load template (invalid uuid)", tid)
				}
			}
		}
		fmt.Println("Requested payload:")
		fmt.Println(modelObject)
		if op == wimark.OperationUpdate {
			db_model := db_results[0]
			j := []byte{}
			switch opt {
			case "wlan":
				wmodel := wimark.WLAN{}
				b, _ := bson.Marshal(db_model)
				bson.Unmarshal(b, &wmodel)
				j, _ = json.Marshal(wmodel)
			case "radius":
				wmodel := wimark.Radius{}
				b, _ := bson.Marshal(db_model)
				bson.Unmarshal(b, &wmodel)
				j, _ = json.Marshal(wmodel)
			case "cpe":
				wmodel := wimark.CPE{}
				b, _ := bson.Marshal(db_model)
				bson.Unmarshal(b, &wmodel)
				j, _ = json.Marshal(wmodel)
			case "cpe_config_templates":
				wmodel := wimark.ConfigRule{}
				b, _ := bson.Marshal(db_model)
				bson.Unmarshal(b, &wmodel)
				j, _ = json.Marshal(wmodel)
			}
			jmap := map[string]interface{}{}
			json.Unmarshal(j, &jmap)
			mergo.Merge(&jmap, modelObject, mergo.WithOverride)
			modelObject = jmap
		}
		payload = map[string]interface{}{
			opt: map[string]interface{}{
				oid: modelObject,
			},
		}
	} else {
		payload = map[string]interface{}{
			opt: map[string]interface{}{
				"uuid": []string{oid},
			},
		}
	}

	if opt == "cpe_config_templates" {
		return doCPECondigRule(op, oid, modelObject)
	}
	reqID := xid.New().String()
	topic := &wimark.RequestTopic{
		SenderModule:   wimark.ModuleCLI,
		SenderID:       "",
		ReceiverModule: wimark.ModuleConfig,
		ReceiverID:     "",
		RequestID:      reqID,
		Operation:      op,
		Tag:            "",
	}

	err := wimark.MQTTPublishMsg(brokerClient, wimark.MQTTDocumentMessage{
		T: topic,
		D: payload,
		R: false,
	})

	fmt.Println("MQTT tx message on topic", topic.TopicPath())

	if err != nil {
		return err
	}

	err = sendMQTTEvent(brokerClient, wimark.SystemEventTypeLocationCacheReload, "reload after "+op.String()+" "+opt, nil)

	return err
}

func doCommand(cmd Command) {
	for opt, fpairs := range cmd.Options {

		switch cmd.Operation {
		case "read":
			fields := []string{}
			for _, f := range fpairs {
				if f.Field != "" {
					s := strings.Split(f.Field, ".")
					fields = append(fields, s...)
				}
			}
			if len(fields) > 0 {
				fields = append(fields, "_id")
			}
			raw_result := dbRead(opt, cmd.ObjectIDs[opt], fields)
			result := make([]map[string]interface{}, len(raw_result))
			for i, v := range raw_result {
				result[i] = fielddig(v, fields)
			}
			for _, e := range result {
				if cfg.PrettyPrint {
					f := strings.Repeat("=", len(e["_id"].(string))+4)
					fmt.Printf("%s\nID: %s\n%s\n", f, e["_id"], f)
					printResultPretty(e, "", opt)
				} else {
					fmt.Printf("id=%s\n", e["_id"])
					printResult(e, "", opt)
				}
				fmt.Printf("\n")
			}
		case "set":
			if len(cmd.ObjectIDs[opt]) > 1 {
				fmt.Println("Too many OIDs!")
				continue
			} else if len(cmd.ObjectIDs[opt]) == 0 {
				fmt.Println("No OIDs!")
				continue
			}
			err := doMQTTRequest(wimark.OperationUpdate, opt, cmd.ObjectIDs[opt][0], cmd.Options[opt])
			if err != nil {
				fmt.Println("error calling mqtt_set:", err)
			}
		case "delete":
			if len(cmd.ObjectIDs[opt]) > 1 {
				fmt.Println("Too many OIDs!")
				continue
			} else if len(cmd.ObjectIDs[opt]) == 0 {
				fmt.Println("No OIDs!")
				continue
			}
			err := doMQTTRequest(wimark.OperationDelete, opt, cmd.ObjectIDs[opt][0], cmd.Options[opt])
			if err != nil {
				fmt.Println("error calling mqtt_delete:", err)
			}
		}
	}
}
