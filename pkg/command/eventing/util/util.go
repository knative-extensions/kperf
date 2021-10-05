// Copyright 2020 The Knative Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/Shopify/sarama"
)

type AdapterSASL struct {
	Enable   bool   `envconfig:"KAFKA_NET_SASL_ENABLE" required:"false" default:"false"`
	User     string `envconfig:"KAFKA_NET_SASL_USER" required:"false"`
	Password string `envconfig:"KAFKA_NET_SASL_PASSWORD" required:"false"`
	//Type     string `envconfig:"KAFKA_NET_SASL_TYPE" required:"false"`
}

type AdapterTLS struct {
	Enable bool `envconfig:"KAFKA_NET_TLS_ENABLE" required:"false"`
	// Cert   string `envconfig:"KAFKA_NET_TLS_CERT" required:"false"`
	// Key    string `envconfig:"KAFKA_NET_TLS_KEY" required:"false"`
	// CACert string `envconfig:"KAFKA_NET_TLS_CA_CERT" required:"false"`
}

type AdapterNet struct {
	SASL AdapterSASL
	TLS  AdapterTLS
}

func UpdateSaramaConfigFromEnv(net AdapterNet, config *sarama.Config) {
	if net.TLS.Enable {
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = &tls.Config{
			MinVersion:               tls.VersionTLS12,
			MaxVersion:               tls.VersionTLS12,
			PreferServerCipherSuites: true,
		}

		log.Printf("Kafka TLS enabled")

		// // if we have TLS, we might want to use the certs for self-signed CERTs
		// if b.auth.TLS.Cacert != "" {
		// 	tlsConfig, err := newTLSConfig(b.auth.TLS.Usercert, b.auth.TLS.Userkey, b.auth.TLS.Cacert)
		// 	if err != nil {
		// 		return nil, fmt.Errorf("Error creating TLS config: %w", err)
		// 	}
		// 	config.Net.TLS.Config = tlsConfig
		// }
	} else {
		log.Printf("Kafka TLS not configured")
	}
	if net.SASL.Enable {
		config.Net.SASL.Enable = true
		config.Net.SASL.Handshake = true

		// if SaslType is not provided we are defaulting to PLAIN
		config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		config.Net.SASL.User = net.SASL.User
		config.Net.SASL.Password = net.SASL.Password
		log.Printf("Configured Kafka auth USER=%s", config.Net.SASL.User)
	} else {
		log.Printf("Auth not configure")
	}
}

//TODO use existing util pkpg

func RequiredGetEnv(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		fmt.Fprintf(os.Stderr, "missing required environment variable %s\n", key)
		os.Exit(1)
	}
	return value
}

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func GetEnvInt(key, fallback string) int {
	value := GetEnv(key, fallback)
	v, err := strconv.Atoi(value)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get int value out of %s for environment variable %s error: %v\n", value, key, err)
		os.Exit(1)
	}
	return v
}

func GetEnvFloat64(key, fallback string) float64 {
	value := GetEnv(key, fallback)
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get int value out of %s for environment variable %s error: %v\n", value, key, err)
		os.Exit(1)
	}
	return v
}
