# All the possible fields for options
[Go back](./README.md)

## WLAN
- name
- ssid
- description
- security

### RADIUS section
- nas_id
- nas_port_id
- radius_acct_servers
- radius_acct_interval
- radius_acct_mirroring

### ACL section
- filtermode
- whitelist
- blacklist
- firewall

#### WLAN related specifics
- hidden
- l2isolate
- pmkcaching
- roam80211r
- wmm
- speed_upload
- speed_download

### network section
#### remote tunneling
- tunneling
- proto
- default_tunnel
- peer_address

#### local vlan (if 0 than untag)
- vlan

#### local nat
- nat
- nat_network

#### Portal authorization section
- guest_control

#### clients specifics section
- beeline_accountng_type

#### 802.11k
- ieee80211k
- rrm_neighbor_report
- rrm_beacon_report

#### 802.11v
- ieee80211v
- wnm_sleep_mode
- bss_transition

#### QTECH fields
- rssi_threshold
- band_steering
- ft_over_ds
- load_balancing

#### for wmwdisd - rssi based disconnector
- signal_connect
- signal_stay
- signal_strikes
- signal_poll_time
- signal_drop_reason

#### generate NAS ID (for roaming - will be generated from bssid)
- nas_generate

## RADIUS
- name
- hostname
- auth_port
- acc_port
- secret
- is_local
- is_portal
- dae_client
- dae_secret
- dae_port

## CPE
- name
- connected
- description
- model
- config_status
- last_error
- config
- state
- first_connection
- last_connection
- last_disconnection

### reconfiguration
- config_not_send
- latitude
- longitude

## CPE Template
- name
- description
- model
- cpes
- mac_prefix
- subnet
- template.wlans
- template.cpe_config_template
- template.tags
- template.location
- is_auto
- is_always

## CPE Model
- name
- description
- caps
- firmwares
- version
