module github.com/jenpet/traebeler

go 1.13

require (
	github.com/bobesa/go-domain-util v0.0.0-20190911083921-4033b5f7dd89
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/sirupsen/logrus v1.7.0
	github.com/stretchr/testify v1.7.0
	github.com/traefik/traefik/v2 v2.5.0
	golang.org/x/sys v0.0.0-20220503163025-988cb79eb6c6 // indirect
	gopkg.in/h2non/gock.v1 v1.0.15
	gotest.tools v2.2.0+incompatible
)

// From traefik
replace (
	github.com/abbot/go-http-auth => github.com/containous/go-http-auth v0.4.1-0.20200324110947-a37a7636d23e
	github.com/docker/docker => github.com/docker/engine v1.4.2-0.20191113042239-ea84732a7725
	github.com/go-check/check => github.com/containous/check v0.0.0-20170915194414-ca0bf163426a
	github.com/gorilla/mux => github.com/containous/mux v0.0.0-20181024131434-c33f32e26898
	github.com/mailgun/minheap => github.com/containous/minheap v0.0.0-20190809180810-6e71eb837595
	github.com/mailgun/multibuf => github.com/containous/multibuf v0.0.0-20190809014333-8b6c9a7e6bba
)
