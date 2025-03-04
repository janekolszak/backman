package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	config Config
	once   sync.Once
)

type Config struct {
	LogLevel           string `json:"log_level"`
	LoggingTimestamp   bool   `json:"logging_timestamp"`
	Username           string
	Password           string
	DisableWeb         bool               `json:"disable_web"`
	DisableMetrics     bool               `json:"disable_metrics"`
	UnprotectedMetrics bool               `json:"unprotected_metrics"`
	Notifications      NotificationConfig `json:"notifications"`
	S3                 S3Config
	Services           map[string]ServiceConfig
	Foreground         bool
}

type S3Config struct {
	DisableSSL          bool   `json:"disable_ssl"`
	SkipSSLVerification bool   `json:"skip_ssl_verification"`
	ServiceLabel        string `json:"service_label"`
	ServiceName         string `json:"service_name"`
	BucketName          string `json:"bucket_name"`
	EncryptionKey       string `json:"encryption_key"`
}

type ServiceConfig struct {
	Schedule  string
	Timeout   TimeoutDuration
	Retention struct {
		Days  int
		Files int
	}
	DirectS3                bool     `json:"direct_s3"`
	DisableColumnStatistics bool     `json:"disable_column_statistics"`
	LogStdErr               bool     `json:"log_stderr"`
	ForceImport             bool     `json:"force_import"`
	LocalBackupPath         string   `json:"local_backup_path"`
	BackupOptions           []string `json:"backup_options"`
	RestoreOptions          []string `json:"restore_options"`
}

type NotificationConfig struct {
	Teams TeamsNotificationConfig `json:"teams,omitempty"`
}

type TeamsNotificationConfig struct {
	Webhook string   `json:"webhook"`
	Events  []string `json:"events"`
}

type TimeoutDuration struct {
	time.Duration
}

func (td TimeoutDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(td.String())
}

func (td *TimeoutDuration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		td.Duration = time.Duration(value)
		return nil
	case string:
		var err error
		td.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
		return nil
	default:
		return errors.New("invalid duration")
	}
}

func Get() *Config {
	once.Do(func() {
		// initialize
		config = Config{
			Services: make(map[string]ServiceConfig),
		}

		// first load config file, if it exists
		if _, err := os.Stat("config.json"); err == nil {
			data, err := ioutil.ReadFile("config.json")
			if err != nil {
				log.Println("could not load 'config.json'")
				log.Fatalln(err.Error())
			}
			if err := json.Unmarshal(data, &config); err != nil {
				log.Println("could not parse 'config.json'")
				log.Fatalln(err.Error())
			}
		}

		// now load & overwrite with env provided config, if it exists
		env := os.Getenv("BACKMAN_CONFIG")
		if len(env) > 0 {
			envConfig := Config{}
			if err := json.Unmarshal([]byte(env), &envConfig); err != nil {
				log.Println("could not parse environment variable 'BACKMAN_CONFIG'")
				log.Fatalln(err.Error())
			}

			// merge config values
			if len(envConfig.LogLevel) > 0 {
				config.LogLevel = envConfig.LogLevel
			}
			if envConfig.LoggingTimestamp {
				config.LoggingTimestamp = envConfig.LoggingTimestamp
			}
			if len(envConfig.Username) > 0 {
				config.Username = envConfig.Username
			}
			if len(envConfig.Password) > 0 {
				config.Password = envConfig.Password
			}
			if envConfig.DisableWeb {
				config.DisableWeb = envConfig.DisableWeb
			}
			if envConfig.DisableMetrics {
				config.DisableMetrics = envConfig.DisableMetrics
			}
			if envConfig.UnprotectedMetrics {
				config.UnprotectedMetrics = envConfig.UnprotectedMetrics
			}
			if len(envConfig.Notifications.Teams.Webhook) > 0 {
				config.Notifications.Teams.Webhook = envConfig.Notifications.Teams.Webhook
			}
			if len(envConfig.Notifications.Teams.Events) > 0 {
				config.Notifications.Teams.Events = envConfig.Notifications.Teams.Events
			}
			if envConfig.S3.DisableSSL {
				config.S3.DisableSSL = envConfig.S3.DisableSSL
			}
			if envConfig.S3.SkipSSLVerification {
				config.S3.SkipSSLVerification = envConfig.S3.SkipSSLVerification
			}
			if len(envConfig.S3.ServiceLabel) > 0 {
				config.S3.ServiceLabel = envConfig.S3.ServiceLabel
			}
			if len(envConfig.S3.ServiceName) > 0 {
				config.S3.ServiceName = envConfig.S3.ServiceName
			}
			if len(envConfig.S3.BucketName) > 0 {
				config.S3.BucketName = envConfig.S3.BucketName
			}
			if len(envConfig.S3.EncryptionKey) > 0 {
				config.S3.EncryptionKey = envConfig.S3.EncryptionKey
			}
			for serviceName, serviceConfig := range envConfig.Services {
				mergedServiceConfig := config.Services[serviceName]
				if len(serviceConfig.Schedule) > 0 {
					mergedServiceConfig.Schedule = serviceConfig.Schedule
				}
				if serviceConfig.Timeout.Seconds() > 1 {
					mergedServiceConfig.Timeout = serviceConfig.Timeout
				}
				if serviceConfig.Retention.Days > 0 {
					mergedServiceConfig.Retention.Days = serviceConfig.Retention.Days
				}
				if serviceConfig.Retention.Files > 0 {
					mergedServiceConfig.Retention.Files = serviceConfig.Retention.Files
				}
				if serviceConfig.DirectS3 {
					mergedServiceConfig.DirectS3 = serviceConfig.DirectS3
				}
				if serviceConfig.DisableColumnStatistics {
					mergedServiceConfig.DisableColumnStatistics = serviceConfig.DisableColumnStatistics
				}
				if serviceConfig.LogStdErr {
					mergedServiceConfig.LogStdErr = serviceConfig.LogStdErr
				}
				if serviceConfig.ForceImport {
					mergedServiceConfig.ForceImport = serviceConfig.ForceImport
				}
				if len(serviceConfig.LocalBackupPath) > 0 {
					mergedServiceConfig.LocalBackupPath = serviceConfig.LocalBackupPath
				}
				if len(serviceConfig.BackupOptions) > 0 {
					mergedServiceConfig.BackupOptions = serviceConfig.BackupOptions
				}
				if len(serviceConfig.RestoreOptions) > 0 {
					mergedServiceConfig.RestoreOptions = serviceConfig.RestoreOptions
				}
				config.Services[serviceName] = mergedServiceConfig
			}
		}

		// ensure we have default values
		if len(config.LogLevel) == 0 {
			config.LogLevel = "info"
		}
		if len(config.S3.ServiceLabel) == 0 {
			config.S3.ServiceLabel = "dynstrg"
		}

		// use username & password from env if defined
		if os.Getenv(BackmanUsername) != "" {
			config.Username = os.Getenv(BackmanUsername)
		}
		if os.Getenv(BackmanPassword) != "" {
			config.Password = os.Getenv(BackmanPassword)
		}

		// use s3 encryption key from env if defined
		if os.Getenv(BackmanEncryptionKey) != "" {
			config.S3.EncryptionKey = os.Getenv(BackmanEncryptionKey)
		}

		// use teams webhook url from env if defined
		if os.Getenv(BackmanTeamsWebhook) != "" {
			config.Notifications.Teams.Webhook = os.Getenv(BackmanTeamsWebhook)
		}

		// use teams events configuration from env if defined
		if os.Getenv(BackmanTeamsEvents) != "" {
			var events []string
			eventsString := os.Getenv(BackmanTeamsEvents)
			if eventsString != "" {
				events = strings.Split(eventsString, ",")
			}

			config.Notifications.Teams.Events = events
		}
	})
	return &config
}
