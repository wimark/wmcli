# wmcli 

Command Line interface for Wimark One (R). 

## structure

wmcli (will) provide such structure of cli:

```
> wmcli [operation] <option> <option object id> <option fields>

# print all objects in collection wlan and its fields/values
> wmcli read wlan

# print field/value of all objects in collection wlan
> wmcli read wlan [field]

# print exact object from collection wlan fields/values
> wmcli read wlan [object_id]

# print exact object from collection wlan field/value
> wmcli read wlan [object_id] [field]

# print exact objects from collection wlan fields/values

> wmcli read wlan [object_id] [object_id] [field] [field] [field]


# sets (or creates if there is no such [object_id]) wlan option fields
# name to Test and ssid field to Test_Wifi and cpe option field
# name to Test_CPE for object [object_id]
> wmcli set wlan [object_id] name="Test" ssid="Test_Wifi" cpe [object_id] name="Test_CPE"
> ... 

# create default object [object_id] to colelction wlan
> wmcli set wlan [object_id]
> ...

# delete object [object_id] from collection wlan
> wmcli delete wlan [object_id]
> ...

# delete wlan option field name value (fallback to default)
> wmcli delete wlan [object_id] name
> ...
```

For operations we have next options:

- read
- set
- delete

For objects options are:

- wlan
- radius
- cpe
- cpe_template
- cpe_model

And for each options fields will be provided according to libwimark one's  
(can be easily deprecated, leaved [here](./FIELDS.md) as example)

## how it works

At first (before launch), wmcli needs a configuration file (config.y(a)ml) or environment configuration with the next parameters:

- MongoDB URL (MONGO_ADDR for env, mongo_addr for yaml)
- MQTT address (MQTT_ADDR for env, mqtt_addr for yaml)

For several operations, wmcli works directly with DB via commands and queries. 
For updating objects, that needed to be updated with wimark's configurator MQTT
is used. 

```

wmcli ---- MQTT ---- configurator app
  |
  |
  DB

```

If provided connection will (in the future) need a connection to the backend with HTTP API - it 
will be realized. For current operations, it is assumed to be unnecessary.

## copyright

Made by Wimark Systems in late 2021-2022.
