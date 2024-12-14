# telegraf-loxone
A Telegraf input plugin for collecting events from Loxone home automation systems.

[![version](https://img.shields.io/badge/status-alpha-red.svg)](https://github.com/andrasg/telegraf-loxone)


## Overview
The plugin allows metrics to be collected by telegraf from Loxone. It uses a websocket connection to Loxone. All changes to the configured UUID's are captured, no aggregation is happening.

Note, only numeric type UUID values are supported.

Note2, only those values will be captured that have the `Use in User Interface` checkbox set in Loxone Config.

## Installation

Download the repo

    git clone https://github.com/andrasg/telegraf-loxone.git

Build

    ./build.sh

Use the output executable. Make sure the file is executable (`chmod +x` as needed). You can use this input plugin using the `execd` plugin, using the following syntax:

```
[[inputs.execd]]
  command = ["/plugin/telegraf-loxone", "-config", "/plugin/telegraf-loxone.conf"]
  signal = "none"
```

Reference the config file that you will be creating in the next section.

## Configuration

Create a configuration file, see `plugin.conf` as an example. Important, that this config file should not be put in the `telegraf.d` folder!

Set `url`, `user` and `key` in the TOML plugin config according to your Loxone settings. 

Define items to be collected in the following format:
```toml
  [[inputs.loxone.item]]
    uuid = "11111111-2222-abcd-ffffffffffffffff"
    bucket = "bucketname"
    measurement = "measurement"
    field = "fieldname"

    ## optional tags
    [inputs.loxone.item.tags]
      tagname = "tagvalue"
```

Tip: you can get the relevant UUID's by opening your *.Loxone file using an XML editor and looking for the relevant GUID's.

If the bucket setting is set, the value will be added as the `bucket` tag to the datapoints. If it is omitted, the output plugin will determine what bucket to write to.

Tip: Use this in combination with the Influx v2 output plugin and set the `bucket_tag` to `bucket` and `exclude_bucket_tag` to `true` in the `[outputs.influx_v2]` section of the telegraf config.

If the `field` setting is missing, the field will be named `value` in the datapoint.

## Troubleshooting

Some more logs are emitted if the `LOGLEVEL` environmental variable is set to `INFO`, `DEBUG` or `TRACE`.

## Acknowledgements

Built using XciD's [Loxone Websocket Golang](https://github.com/XciD/loxone-ws) package.