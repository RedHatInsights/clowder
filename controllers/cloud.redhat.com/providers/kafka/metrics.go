package kafka

var metricsData = []byte(`{
    "metrics": {
      "lowercaseOutputName": true,
      "rules": [
        {
          "labels": {
            "clientId": "$3",
            "partition": "$5",
            "topic": "$4"
          },
          "name": "kafka_server_$1_$2",
          "pattern": "kafka.server<type=(.+), name=(.+), clientId=(.+), topic=(.+), partition=(.*)><>Value",
          "type": "GAUGE"
        },
        {
          "labels": {
            "broker": "$4:$5",
            "clientId": "$3"
          },
          "name": "kafka_server_$1_$2",
          "pattern": "kafka.server<type=(.+), name=(.+), clientId=(.+), brokerHost=(.+), brokerPort=(.+)><>Value",
          "type": "GAUGE"
        },
        {
          "labels": {
            "cipher": "$5",
            "listener": "$2",
            "networkProcessor": "$3",
            "protocol": "$4"
          },
          "name": "kafka_server_$1_connections_tls_info",
          "pattern": "kafka.server<type=(.+), cipher=(.+), protocol=(.+), listener=(.+), networkProcessor=(.+)><>connections",
          "type": "GAUGE"
        },
        {
          "labels": {
            "clientSoftwareName": "$2",
            "clientSoftwareVersion": "$3",
            "listener": "$4",
            "networkProcessor": "$5"
          },
          "name": "kafka_server_$1_connections_software",
          "pattern": "kafka.server<type=(.+), clientSoftwareName=(.+), clientSoftwareVersion=(.+), listener=(.+), networkProcessor=(.+)><>connections",
          "type": "GAUGE"
        },
        {
          "labels": {
            "listener": "$2",
            "networkProcessor": "$3"
          },
          "name": "kafka_server_$1_$4",
          "pattern": "kafka.server<type=(.+), listener=(.+), networkProcessor=(.+)><>(.+):",
          "type": "GAUGE"
        },
        {
          "labels": {
            "listener": "$2",
            "networkProcessor": "$3"
          },
          "name": "kafka_server_$1_$4",
          "pattern": "kafka.server<type=(.+), listener=(.+), networkProcessor=(.+)><>(.+)",
          "type": "GAUGE"
        },
        {
          "name": "kafka_$1_$2_$3_percent",
          "pattern": "kafka.(\\w+)<type=(.+), name=(.+)Percent\\w*><>MeanRate",
          "type": "GAUGE"
        },
        {
          "name": "kafka_$1_$2_$3_percent",
          "pattern": "kafka.(\\w+)<type=(.+), name=(.+)Percent\\w*><>Value",
          "type": "GAUGE"
        },
        {
          "labels": {
            "$4": "$5"
          },
          "name": "kafka_$1_$2_$3_percent",
          "pattern": "kafka.(\\w+)<type=(.+), name=(.+)Percent\\w*, (.+)=(.+)><>Value",
          "type": "GAUGE"
        },
        {
          "labels": {
            "$4": "$5",
            "$6": "$7"
          },
          "name": "kafka_$1_$2_$3_total",
          "pattern": "kafka.(\\w+)<type=(.+), name=(.+)PerSec\\w*, (.+)=(.+), (.+)=(.+)><>Count",
          "type": "COUNTER"
        },
        {
          "labels": {
            "$4": "$5"
          },
          "name": "kafka_$1_$2_$3_total",
          "pattern": "kafka.(\\w+)<type=(.+), name=(.+)PerSec\\w*, (.+)=(.+)><>Count",
          "type": "COUNTER"
        },
        {
          "name": "kafka_$1_$2_$3_total",
          "pattern": "kafka.(\\w+)<type=(.+), name=(.+)PerSec\\w*><>Count",
          "type": "COUNTER"
        },
        {
          "labels": {
            "$4": "$5",
            "$6": "$7"
          },
          "name": "kafka_$1_$2_$3",
          "pattern": "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.+), (.+)=(.+)><>Value",
          "type": "GAUGE"
        },
        {
          "labels": {
            "$4": "$5"
          },
          "name": "kafka_$1_$2_$3",
          "pattern": "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.+)><>Value",
          "type": "GAUGE"
        },
        {
          "name": "kafka_$1_$2_$3",
          "pattern": "kafka.(\\w+)<type=(.+), name=(.+)><>Value",
          "type": "GAUGE"
        },
        {
          "labels": {
            "$4": "$5",
            "$6": "$7"
          },
          "name": "kafka_$1_$2_$3_count",
          "pattern": "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.+), (.+)=(.+)><>Count",
          "type": "COUNTER"
        },
        {
          "labels": {
            "$4": "$5",
            "$6": "$7",
            "quantile": "0.$8"
          },
          "name": "kafka_$1_$2_$3",
          "pattern": "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.*), (.+)=(.+)><>(\\d+)thPercentile",
          "type": "GAUGE"
        },
        {
          "labels": {
            "$4": "$5"
          },
          "name": "kafka_$1_$2_$3_count",
          "pattern": "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.+)><>Count",
          "type": "COUNTER"
        },
        {
          "labels": {
            "$4": "$5",
            "quantile": "0.$6"
          },
          "name": "kafka_$1_$2_$3",
          "pattern": "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.*)><>(\\d+)thPercentile",
          "type": "GAUGE"
        },
        {
          "name": "kafka_$1_$2_$3_count",
          "pattern": "kafka.(\\w+)<type=(.+), name=(.+)><>Count",
          "type": "COUNTER"
        },
        {
          "labels": {
            "quantile": "0.$4"
          },
          "name": "kafka_$1_$2_$3",
          "pattern": "kafka.(\\w+)<type=(.+), name=(.+)><>(\\d+)thPercentile",
          "type": "GAUGE"
        }
      ]
    }
  }`)
