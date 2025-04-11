# telegraf-loxone
A Telegraf input plugin for collecting events from Loxone home automation systems.

[![version](https://img.shields.io/badge/status-alpha-red.svg)](https://github.com/andrasg/telegraf-loxone)


## Overview
The plugin allows metrics to be collected by telegraf from Loxone. It uses a websocket connection to Loxone. All changes to the configured UUID's are captured, no aggregation is happening.

> Note, only numeric type UUID values are supported.

> Note2, Loxone miniserver only emits status updates for those values that have the `Use in User Interface` checkbox set in Loxone Config. To avoid cluttering your Loxone app UI, create a dedicated user for this plugin and set permissions as needed for this user.

## Installation

Download the repo

    git clone https://github.com/andrasg/telegraf-loxone.git

If you don't have Go installed, the best is to open the repo in VS Code using a devcontainer.

Build using:

    ./build.sh

Copy the output executable to where you will be using it from. Make sure the file is executable (`chmod +x` as needed).

### Telegraf input configuration

To wire up this plugin to Telegraf, you need a config section/file for telegraf. You can use this input plugin using the `execd` input plugin in Telegraf, using the following syntax:

```
[[inputs.execd]]
  command = ["/plugin/telegraf-loxone", "-config", "/plugin/telegraf-loxone.conf"]
  signal = "none"
```

In this config, reference the plugin config file that you will be creating in the next section.

### Plugin configuration

Create the plugin configuration file, see `plugin.conf` as an example. Important, that this config file is different from the telegraf config file from above. The file `plugin.conf` should not be placed in the `telegraf.d` folder!

Set `url`, `user` and `key` in the TOML plugin config according to your Loxone settings. 

```toml
[[inputs.loxone]]
  ## Loxone IP address
  url = "192.168.1.253"

  ## Username for authentication to Loxone
  user = "user"

  ## password for authentication to Loxone
  key = "pass"
```

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

> Hint: you can get the relevant UUID's by opening your *.Loxone file using an XML editor and looking for the relevant GUID's.

If the bucket setting is set, the value will be added as the `bucket` tag to the datapoints. If it is omitted, the output plugin will determine what bucket to write to.

> Hint: Use this in combination with the Influx v2 output plugin and set the `bucket_tag` to `bucket` and `exclude_bucket_tag` to `true` in the `[outputs.influx_v2]` section of the telegraf config.

If the `field` setting is missing, the field will be named `value` in the datapoint.

### Alternative config format

Alternatively, the config can be supplied in JSON file instead of the inline TOML format. Use the following example for crafting the external config file:

```json
{
  "pointMappings": [
    {
      "bucket": "<destination_bucket>",
      "measurements": [
        {
          "name": "<measurement_name>",
          "datapoints":
          [
            {
              "fields":
              {
                "<field_name>": "<uuid>",
                "...": "..."
              },
              "tags": { 
                "<tag1_name>": "<tag1_value>",
                "...": "..."
              }		
            },
            { ... }
          ]
        },
        { ... }
      ]
    },
    { ... }
  ]
}
```

## Troubleshooting

Some more logs are emitted if the `LOGLEVEL` environmental variable is set to `INFO`, `DEBUG` or `TRACE`.

## Acknowledgements

Built using XciD's [Loxone Websocket Golang](https://github.com/XciD/loxone-ws) package.