module jpy-cli

go 1.25

require (
	github.com/gorilla/websocket v1.5.3
	github.com/spf13/cobra v1.10.2
	github.com/vmihailenco/msgpack/v5 v5.4.1
	go.bug.st/serial v1.6.2
)

replace adminApi => cnb.cool/htsystem/adminApi v1.0.3

replace (
	github.com/ghp3000/netclient => ./third_party/ApiAgent/pkg/netclient
	github.com/ghp3000/public => ./third_party/ApiAgent/pkg/public
	github.com/ghp3000/utils => ./third_party/ApiAgent/pkg/utils
	portmap => ./third_party/ApiAgent/pkg/portmap
	socks5 => ./third_party/ApiAgent/pkg/socks5
)

require (
	github.com/creack/goselect v0.1.2 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	golang.org/x/sys v0.5.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
