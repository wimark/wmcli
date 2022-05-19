# Использование wmcli для создания активатора

## Работа с wmcli

### Конфигурация

Для успешного запуска программы необходимо указать адрес базы данных MongoDB и брокера MQTT.
Это можно сделать как через конфигурационный yaml файл (config.yml/config.yaml), так и через переменные окружения.

Также есть возможность сделать "красивый" вывод данных с отступами и использовать для вывода ключи формата базы данных, а не Go структур.

Список всех настроек:

```
WMCLI_MONGO_ADDR=127.0.0.1:27001
WMCLI_MQTT_ADDR=tcp://127.0.0.1:1883
PRETTY_PRINT=false
USE_DB_OUTPUT=false


$ cat config.yaml
mongo_addr: '127.0.0.1:27001'
mqtt_addr: 'tcp://127.0.0.1:1883'
pretty_print: false
db_output: false
```

### Использование

wmcli может работать как в интерактивном режиме с автодополнением, так и в неинтерактивном режиме.
В основе лежат три базовые команды операций:

- `read` - чтение из базы данных объектов в коллекции / всей коллекции, с возможностью фильтрации по полям

- `set` - выставление параметров (создание, в случае отсутствия) объекта в коллекции через запрос по MQTT

- `delete` - удаление объекта из коллекции по MQTT

Доступны 5 коллекций:

- `wlan` - конфигурации Wi-Fi сетей

- `radius` - конфигурации RADIUS

- `cpe` - точки доступа на платформе, имеется специальный ключ `FromTemplate` для применения шаблона

- `cpe_model` - модели точек доступа, только для чтения

- `cpe_template` - шаблоны заполнения точек доступа, _запись производится напрямую в базу данных_

Структура получается следующая:
`wmcli [операция] [коллекция] <uuid объекта> <поле (чтение, удаление)/поле=значение (запись)>...`

В интерактивном режиме новый uuid объекта генерируется в автодополнение.
В неинтерактивном режиме можно воспользоваться uuidgen (e2fsprogs/uuid-runtime, должно быть почти в любом дистрибутиве из коробки).

Примеры использования:

```
 Вывести все объекты в коллекции wlan
> wmcli read wlan

# Вывести поле Name у всех объектов в коллекции wlan
> wmcli read wlan Name

# Вывести все данные объекта [uuid] в коллекции wlan
> wmcli read wlan [uuid]

# Вывести поле Name у объекта [uuid] в коллекции wlan
> wmcli read wlan [uuid] Name

# Вывести поля Name, Description, SSID у объектов [uuid1] и [uuid2] в коллекции wlan
> wmcli read wlan [uuid1] [uuid2] Name Description SSID


# Создать или обновить данные объекта [uuid] с полями name и ssid в коллекции wlan
> wmcli set wlan [uuid] Name="Test" SSID="Test_Wifi"
> ... 

# Создать новый объект [uuid] в коллекции wlan
> wmcli set wlan [uuid]
> ...

# Удалить объект [uuid] из коллекции wlan
> wmcli delete wlan [uuid]
> ...
```

В интерактивном режиме в автодополнении присутствуют подсказки типа данных поля, их три:

- `string` - заполняется с кавычками: `Field="..."`

- `int` - заполняется без кавычек: `Field=42`

- `bool` - может быть `true` или `false`, без кавычек: `Field=true`

Также, некоторые поля помечены как `[RO]`. Такие поля доступны только для чтения.

Существуют поля, которые заканчиваются на `.#` - это массивы.
Для добавления элементов в массив нужно просто повторить заполнение такого поля, например:

```
set ... Field.#="one" Field.#="two"
read ...
Field.#=["one", "two"]
```

Также есть поля, внутри которых встречается `.*` - это словари, этим символом помечаются поля в автодополнении в интерактивном режиме.
Чтобы задать свое имя ключа, нужно заменить `*` на название ключа, например, `Field.*.Name=...` -> `Field.Key.Name=...`

## Алгоритм работы активатора

#### Создание нового WLAN объекта

Здесь создается объект wlan с минимальным набором необходимых полей.

```
wmcli set wlan 570beb53-5efb-4c0b-927e-0bf6e4e2b42f Name="RTK_TEST_WLAN" SSID="RTK_TEST" Security.wpa2personal.WPACommon.Suites.#="aes" Security.wpa2personal.PSK="12345678" Tunneling=true PeerAddress="10.10.10.10" Proto="gretap"
```

#### Создание нового Radius объекта

Тоже самое с объектом radius'а

```
wmcli set radius fdef05f6-5cf7-4750-b5d7-39677af38d2e Name="RTK_RADIUS_TEST" Hostname="rtk.radius.ru" Acc_port="1813" Auth_port="1812" Is_local=true DaePort="3799" Secret="rtksecret"
```

#### Назначение Radius объекта WLAN'у

Здесь в ранее созданном объекте wlan обновляется поле `RadiusAcctServers.#`, которое является массивом, и в который добавляется первым элементом UUID объекта radius из прошлого шага.

```
wmcli set wlan 570beb53-5efb-4c0b-927e-0bf6e4e2b42f RadiusAcctServers.#="fdef05f6-5cf7-4750-b5d7-39677af38d2e"
```

#### Создание шаблона для точки с вписанным WLAN объектом

Просто создается объект cpe_template с необходимыми для точки полями.

```
wmcli set cpe_template ebd7b67e-e8ae-45ce-bba7-7cfa7ede05bd mac_prefix="C0:A6:6D:00:32:20" name="TestRTK" description="ADDRESS" template.cpe_config_template.wifi.radio0.require_mode="false" template.cpe_config_template.wifi.*.country="RU" template.cpe_config_template.wifi.radio0.bandmode="11a" template.cpe_config_template.wifi.radio0.bandwidth="HT20" template.cpe_config_template.wifi.radio0.power.range.#=0 template.cpe_config_template.wifi.radio0.power.range.#=10 template.cpe_config_template.wifi.radio0.frequency="2.4" template.cpe_config_template.wifi.radio0.require_mode="false" template.cpe_config_template.wifi.*.country="RU" template.cpe_config_template.wifi.radio1.bandmode="11g" template.cpe_config_template.wifi.radio1.bandwidth="HT20" template.cpe_config_template.wifi.radio1.power.range.#=0 template.cpe_config_template.wifi.radio1.power.range.#=10 template.cpe_config_template.wifi.radio1.frequency="5" template.cpe_config_template.wifi.radio0.wlans.#="570beb53-5efb-4c0b-927e-0bf6e4e2b42f" template.cpe_config_template.wifi.radio1.wlans.#="570beb53-5efb-4c0b-927e-0bf6e4e2b42f"
```

#### "Отлов" точки по MAC адресу, для получения ее UUID

Здесь используется скрипт, код которого будет представлен ниже, и на базе которого можно получить представление о том как работает wmcli в скриптинге.
В нем получается список всех точек доступа из коллекции cpe с фильтрацией по полю State[RO].Network.MACAddr, в котором указан MAC адрес точки.
Затем он приводится в удобный для парсинга вид, и по нему проходится grep с поиском нужного адреса из аргумента к скрипту.

```
./id_by_mac.sh C0:A6:6D:00:32:20
```

#### Применение шаблона на UUID точки

Здесь в техническое поле пойманной точки вносится UUID шаблона, созданного ранее.
После этого формируется запрос с обновлением полей точки по данным из шаблона.

```
wmcli set cpe bcaad286-14fa-9cf1-f8bd-c0a66d003220 FromTemplate="ebd7b67e-e8ae-45ce-bba7-7cfa7ede05bd"
```

## Скрипт для получения UUID точек по их MAC адресу

```bash
#!/bin/bash

# export needed variables
export WMCLI_MQTT_ADDR=tcp://127.0.0.1:1883
export WMCLI_MONGO_ADDR=127.0.0.1:57017
export WMCLI_PRETTY_PRINT=false
BIN=./wmcli

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
```
