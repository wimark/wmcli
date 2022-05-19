#!/bin/bash

# export needed variables
export WMCLI_MQTT_ADDR=tcp://{platformaddr}:1883
export WMCLI_MONGO_ADDR={platformaddr}:57017
export WMCLI_PRETTY_PRINT=false
BIN=./main

cpe_id_mac_list() {
    # filter output by macaddr
    CPE_ID_MAC=$($BIN read cpe State[RO].Network.MACAddr)
    
    # split output for id:mac pairs
    # replace double newline separator with tab
    CPE_ID_MAC=${CPE_ID_MAC//$'\n'$'\n'/$'\t'}  
    
    # filter output by tab ifs 
    oifs=$IFS
    IFS=$'\t'
    CPE_ID_MAC=$(for i in $CPE_ID_MAC
    do
        i=${i//"id="/}
        i=${i//"State.VLAN.MACAddr="/}
        echo ${i//$'\n'/' '}
    done
    )
    # restore ifs
    IFS=$oifs
    
    echo "$CPE_ID_MAC"
}

get_id_by_mac() {
    cpe_id_mac_list | grep $1 | cut -d' ' -f1
}

get_id_by_mac $1
