package config

type Config struct {
	RabbitMQHost string
}

func NewConfig() *Config {
	return &Config{
		RabbitMQHost: "localhost",
	}
}
