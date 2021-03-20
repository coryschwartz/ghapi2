package main

import (
	"os"
	"time"

	"github.com/gregjones/httpcache"
	"github.com/palantir/go-baseapp/baseapp"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/rs/zerolog"
	"goji.io/pat"
)

type Config struct {
	Server    baseapp.HTTPConfig
	Github    githubapp.Config
	AppConfig MyApplicationConfig
}

type MyApplicationConfig struct {
	PullRequestPreamble string
}

func main() {
	config := Config{
		Server: baseapp.HTTPConfig{
			Address: "127.0.0.1",
			Port:    3000,
		},
		Github: githubapp.Config{
			V4APIURL: "https://api.github.com/",
			App: struct {
				IntegrationID int64  `yaml:"integration_id" json:"integrationId"`
				WebhookSecret string `yaml:"webhook_secret" json:"webhookSecret"`
				PrivateKey    string `yaml:"private_key" json:"privateKey"`
			}{106087,
				"supersecret",
				`-----BEGIN RSA PRIVATE KEY-----
MIIEpgIBAAKCAQEA9EWyAInkZAF1FJddbchIlzXHTcrrupToasUo6DwZY/cSkq1V
40RqHVMvk8VCHBl9CCY1Ym60znZgfoG5EgsEhpOXRCs5lgiLq2JVfX4UkIF42nMs
iuhZyLWLEoR1RMgVK/zKyT0jvEiruxHg2pxMOzJfQrsQV2IfeY9TYOrTRfyF07sB
/iMEA3X4J3y23r5MIvszJi5THYieFDrO3UotOqJZMwdYOmNBgy2fKh4HoO46YBU7
APrJBW+7XRNa1QbtTMBEpC/BYdyVBlCT7PKEhXFpOh2ZJvy3QtuyViVXQ+z+h9Nm
LngAqXtdpAN2pLvMKm8O/qnYLyIfyXcMXXPHKwIDAQABAoIBAQDOkUWnGPTv8R+O
azZSCUYBwTOqwIg5/4TQDay7P1+FXsHxEe4Iw6ks5VTdlLmEQ7WtN4p7k/0If6i4
MoFMFc8c8yC/QAJxswZRx1VeS0meri6CJVWsnjKW/Zb+8M6ufLkSurLOHQrkRVwc
VVEd7YC1qrJOHx0BmHPfe2naEprZmbyoTT5ICTnFusULv4/SefYDVhBXhYyHJ94U
MDbuID+3M0Y4NCNl9yJ15HYSgv14E4GpQdLAPui6dWTqEsXafPZHzsDccUg/vhW4
2KdIiDiEcrRvkZl71Y+7FT3pDXOgJfLIN+5egqFOYAKWv/fdcm99gDPCLYUTjXT7
KoWhbUj5AoGBAPxkmKuNZ3WACixcxhlW1Ree9pgDMOMVLZbbw31VZEPPlmktOsLm
Hx6Tx0+En4MCpbs6gzaqW2djuTCuckKQpXl/4CMlLIMK/swlXANZJB4e5+XY06tl
bxU8Q5bvR75NC7Dl4Ocjbtx67Y9zlh1vjgoAS9O56FaYJIk19iVK9sP/AoGBAPfD
Y3mJIPxW4V+4DIxPdftbQzYaZOltHw+SYcA98a6laiv0U+ZOwyJq7SQIP16E2G6I
WqWOk40/vVD8gECxat4Ij1Q0ZfT8i7teMGMjN0FdjBT+eZbacPRSpjNBAnu0I7DH
MD9aCaVK8SaEtM3xIdFx9nmaTnuOi1NTwaQ4DUzVAoGBAO+Xe0pXSKBFNOMaCr/h
KxZqQ8LYPJ9E6mssIa6n0i+BL1KWqhJ8K4x2Up0M0/OlHrjWedr56x0BkLpCz5qa
/0qQdrBGSLP5Sxl2WZugEmY5hoAtzfoFp2asN6lfamafcvqxrkcc3s+ULlGgMx+s
V1TtJQ5Pi9wwP3a1b/3E5O33AoGBAO7OpVK/mcue8hwQigezj0R28pFzX1CenRGl
RhLFoe10AqHbHgMeZ3cFGQ1h5bJ02Sewxa5NfmrmxNMKjZPNbfQUzBGdb6hywzwV
zQ3BI8EdKagSn5+HbNgR0aAVSQ9y0fPSCe8GGcX4NfeqcZsOkEoQTkFnOb5d5myq
jjp6zFcBAoGBAJKlsKn/uLBwq1UyXOGwyY172HPxWuEtJQNFO/xMmKTNchlbDulg
a/VY6DJyDLpuRXH80wWl4h6J1Cf71QoydFb5SnRZb9gXZeftNbbUjirr6elG3XZ4
pQV+tHntQb5aLBeiM2jr46LF42CKPIzSd3o2/1QbD9l08q6HSmdt0it0
-----END RSA PRIVATE KEY-----
`},
		},
		AppConfig: MyApplicationConfig{
			PullRequestPreamble: "I am echo bot",
		},
	}

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	server, err := baseapp.NewServer(
		config.Server,
		baseapp.DefaultParams(logger, "ghapi2")...,
	)
	if err != nil {
		panic(err)
	}

	cc, err := githubapp.NewDefaultCachingClientCreator(
		config.Github,
		githubapp.WithClientUserAgent("ghapi2/0.0.0"),
		githubapp.WithClientTimeout(5*time.Second),
		githubapp.WithClientCaching(false, func() httpcache.Cache { return httpcache.NewMemoryCache() }),
		githubapp.WithClientMiddleware(
			githubapp.ClientMetrics(server.Registry()),
		),
	)

	if err != nil {
		panic(err)
	}

	prCommentHandler := &PRCommentHandler{
		ClientCreator: cc,
		preamble:      config.AppConfig.PullRequestPreamble,
	}
	webHookHandler := githubapp.NewDefaultEventDispatcher(config.Github, prCommentHandler)
	server.Mux().Handle(pat.Post(githubapp.DefaultWebhookRoute), webHookHandler)

	err = server.Start()
	if err != nil {
		panic(err)
	}

	// change
	// Another change
	// Grr
}
