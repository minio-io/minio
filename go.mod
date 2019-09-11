module github.com/minio/minio

go 1.12

require (
	cloud.google.com/go v0.37.2
	contrib.go.opencensus.io/exporter/ocagent v0.5.0 // indirect
	github.com/Azure/azure-sdk-for-go v27.0.0+incompatible
	github.com/Azure/go-autorest v11.7.0+incompatible
	github.com/DataDog/zstd v1.4.0 // indirect
	github.com/alecthomas/participle v0.2.1
	github.com/aliyun/aliyun-oss-go-sdk v0.0.0-20190307165228-86c17b95fcd5
	github.com/aws/aws-sdk-go v1.20.21
	github.com/bcicen/jstream v0.0.0-20190220045926-16c1f8af81c2
	github.com/cheggaaa/pb v1.0.28
	github.com/coredns/coredns v1.4.0
	github.com/coreos/bbolt v1.3.3 // indirect
	github.com/coreos/etcd v3.3.12+incompatible
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/djherbis/atime v1.0.0
	github.com/dustin/go-humanize v1.0.0
	github.com/eapache/go-resiliency v1.2.0 // indirect
	github.com/eclipse/paho.mqtt.golang v1.1.2-0.20190322152051-20337d8c3947
	github.com/elazarl/go-bindata-assetfs v1.0.0
	github.com/fatih/color v1.7.0
	github.com/fatih/structs v1.1.0
	github.com/go-sql-driver/mysql v1.4.1
	github.com/golang/groupcache v0.0.0-20190702054246-869f871628b6 // indirect
	github.com/gomodule/redigo v2.0.0+incompatible
	github.com/gorilla/handlers v1.4.0
	github.com/gorilla/mux v1.7.0
	github.com/gorilla/rpc v1.2.0+incompatible
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.9.0 // indirect
	github.com/hashicorp/go-rootcerts v1.0.1 // indirect
	github.com/hashicorp/raft v1.1.0 // indirect
	github.com/hashicorp/vault v1.1.0
	github.com/inconshreveable/go-update v0.0.0-20160112193335-8152e7eb6ccf
	github.com/klauspost/compress v1.8.3
	github.com/klauspost/cpuid v1.2.1 // indirect
	github.com/klauspost/pgzip v1.2.1
	github.com/klauspost/readahead v1.3.0
	github.com/klauspost/reedsolomon v1.9.1
	github.com/kurin/blazer v0.5.4-0.20190613185654-cf2f27cc0be3
	github.com/lib/pq v1.0.0
	github.com/mattn/go-isatty v0.0.7
	github.com/miekg/dns v1.1.8
	github.com/minio/cli v1.21.0
	github.com/minio/dsync/v2 v2.0.0
	github.com/minio/gokrb5/v7 v7.2.5
	github.com/minio/hdfs/v3 v3.0.1
	github.com/minio/highwayhash v1.0.0
	github.com/minio/lsync v1.0.1
	github.com/minio/mc v0.0.0-20190529152718-f4bb0b8850cb
	github.com/minio/minio-go v0.0.0-20190327203652-5325257a208f
	github.com/minio/minio-go/v6 v6.0.29
	github.com/minio/parquet-go v0.0.0-20190318185229-9d767baf1679
	github.com/minio/sha256-simd v0.1.0
	github.com/minio/sio v0.2.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/nats-io/go-nats-streaming v0.4.4 // indirect
	github.com/nats-io/nats-server v1.4.1 // indirect
	github.com/nats-io/nats-streaming-server v0.14.2 // indirect
	github.com/nats-io/nats.go v1.8.0
	github.com/nats-io/stan.go v0.4.5
	github.com/ncw/directio v1.0.5
	github.com/nsqio/go-nsq v1.0.7
	github.com/pkg/errors v0.8.1
	github.com/pkg/profile v1.3.0
	github.com/prometheus/client_golang v0.9.3-0.20190127221311-3c4408c8b829
	github.com/rcrowley/go-metrics v0.0.0-20190704165056-9c2d0518ed81 // indirect
	github.com/rjeczalik/notify v0.9.2
	github.com/rs/cors v1.6.0
	github.com/sirupsen/logrus v1.4.2
	github.com/skyrings/skyring-common v0.0.0-20160929130248-d1c0bb1cbd5e
	github.com/streadway/amqp v0.0.0-20190402114354-16ed540749f6
	github.com/tidwall/gjson v1.2.1
	github.com/tidwall/pretty v1.0.0 // indirect
	github.com/tidwall/sjson v1.0.4
	github.com/valyala/fastjson v1.4.1
	github.com/valyala/tcplisten v0.0.0-20161114210144-ceec8f93295a
	go.etcd.io/bbolt v1.3.3 // indirect
	go.uber.org/atomic v1.3.2
	golang.org/x/crypto v0.0.0-20190829043050-9756ffdc2472
	golang.org/x/lint v0.0.0-20190909230951-414d861bb4ac // indirect
	golang.org/x/net v0.0.0-20190827160401-ba9fcec4b297 // indirect
	golang.org/x/sys v0.0.0-20190904154756-749cb33beabd
	golang.org/x/tools v0.0.0-20190911174233-4f2ddba30aff // indirect
	google.golang.org/api v0.4.0
	gopkg.in/Shopify/sarama.v1 v1.20.0
	gopkg.in/olivere/elastic.v5 v5.0.80
	gopkg.in/yaml.v2 v2.2.2
)

// Added for go1.13 migration https://github.com/golang/go/issues/32805
replace github.com/gorilla/rpc v1.2.0+incompatible => github.com/gorilla/rpc v1.2.0
