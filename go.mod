module go_zhhx_framework

go 1.19

require (
	github.com/sirupsen/logrus v1.9.0
	github.com/zhhx/tcp_framework/tcp_client v0.0.0
	github.com/zhhx/zhhx_event v0.0.0
)

require golang.org/x/sys v0.0.0-20220715151400-c0bba94af5f8 // indirect

replace github.com/zhhx/tcp_framework/tcp_client => ./tcp_framework/tcp_client

replace github.com/zhhx/zhhx_event => ./zhhx_event
