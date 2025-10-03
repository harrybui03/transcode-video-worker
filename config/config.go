package config

import (
	"database/sql"
	_ "github.com/lib/pq"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/viper"
)

type Config struct {
	MinIOBucket string        `yaml:"minio_bucket"`
	App         App           `yaml:"app"`
	DB          *sql.DB       `yaml:"db"`
	Queue       *RabbitMQ     `yaml:"rabbitmq"`
	Storage     *minio.Client `yaml:"storage"`
	Server      Server        `yaml:"server"`
}

type App struct {
	Environment string `yaml:"environment"`
	Host        string `yaml:"host"`
	Protocol    string `yaml:"protocol"`
}

type Server struct {
	HttpPort string `yaml:"http_port"`
	Workers  int    `yaml:"workers"`
}

type RabbitMQ struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	User         string `json:"user"`
	Pass         string `json:"pass"`
	ExchangeName string `json:"exchange_name"`
	Kind         string `json:"kind"`
}

func Load(path string) (*Config, error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("postgres", viper.GetString("postgresql_host"))
	if err != nil {
		return nil, err
	}

	rabbitmq := &RabbitMQ{
		Host: viper.GetString("rabbitmq_host"),
		Port: viper.GetInt("rabbitmq_port"),
		User: viper.GetString("rabbitmq_user"),
		Pass: viper.GetString("rabbitmq_pass"),
		Kind: viper.GetString("rabbitmq_kind"),
	}

	minioClient, err := minio.New(viper.GetString("minio.url"), &minio.Options{
		Creds:  credentials.NewStaticV4(viper.GetString("minio.access_id"), viper.GetString("minio.secret_access_key"), ""),
		Secure: false,
	})
	if err != nil {
		return nil, err
	}

	return &Config{
		MinIOBucket: viper.GetString("minio.bucket"),
		App: App{
			Environment: viper.GetString("app.environment"),
			Host:        viper.GetString("app.host"),
			Protocol:    viper.GetString("app.protocol"),
		},
		Server: Server{
			HttpPort: viper.GetString("server.port"),
			Workers:  viper.GetInt("server.workers"),
		},
		DB:      db,
		Queue:   rabbitmq,
		Storage: minioClient,
	}, nil
}
