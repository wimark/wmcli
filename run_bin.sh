[ ! -f main ] && go build cmd/wmcli/*

WMCLI_MQTT_ADDR=tcp://{platformaddr}:1883 \
WMCLI_MONGO_ADDR={platformaddr}:57017 \
./main "$@"
