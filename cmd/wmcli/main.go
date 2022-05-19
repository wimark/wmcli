package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	mongo "github.com/wimark/libmongo"
	wimark "github.com/wimark/libwimark"

	prompt "github.com/c-bata/go-prompt"
	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/cristalhq/aconfig"
	"github.com/cristalhq/aconfig/aconfigyaml"
)

const (
	program = "wmcli"
	version = "0.0.1"
)

type Config struct {
	MongoAddr   string `default:"" env:"MONGO_ADDR" yaml:"mongo_addr"`
	MQTTAddr    string `default:"" env:"MQTT_ADDR" yaml:"mqtt_addr"`
	PrettyPrint bool   `default:"false" env:"PRETTY_PRINT" yaml:"pretty_print"`
	UseDbOutput bool   `default:"true" env:"USE_DB_OUTPUT" yaml:"db_output"`
}

var (
	cfg          Config
	cmd          Command
	db           *mongo.MongoDb
	brokerClient mqtt.Client
)

// Fix for strange Bash behaviour https://github.com/c-bata/go-prompt/issues/228#issuecomment-820639887
func handleExit() {
	rawModeOff := exec.Command("/bin/stty", "-raw", "echo")
	rawModeOff.Stdin = os.Stdin
	_ = rawModeOff.Run()
	rawModeOff.Wait()
}

func connectMongo(addr string, timeout time.Duration) (*mongo.MongoDb, error) {
	var d = mongo.GetDb()
	return d, d.ConnectWithTimeout(addr, timeout)
}

func main() {

	defer handleExit()

	loader := aconfig.LoaderFor(&cfg, aconfig.Config{
		SkipFlags: true,
		EnvPrefix: "WMCLI",
		Files:     []string{"config.yml", "config.yaml"},
		FileDecoders: map[string]aconfig.FileDecoder{
			".yml":  aconfigyaml.New(),
			".yaml": aconfigyaml.New(),
		},
	})

	if err := loader.Load(); err != nil {
		panic(err)
	}

	var err error
	db, err = connectMongo(cfg.MongoAddr, 60)
	if err != nil {
		panic(err)
	}

	brokerClient, err = wimark.MQTTServiceStartWithId(cfg.MQTTAddr, wimark.ModuleCLI,
		wimark.Version{
			Version: version,
			Commit:  "",
			Build:   0,
		},
		newUUID(), nil)
	if err != nil {
		db.Disconnect()
		panic(err)
	}

	rootLocationID = getLocationByName("/").ID

	if len(os.Args) > 1 {
		promptIsInteractive = false
		promptExecNoninteractive(os.Args[1:])
		return
	}

	fmt.Println("Type exit or quit to close.")
	p := prompt.New(
		promptExec,
		promptCompleter,
		prompt.OptionPrefix(">> "),
		prompt.OptionTitle(program),
	)
	p.Run()
}
