package main

type Config struct {
	Database string `config:"DATABASE_NAME" yaml:"database"`
	User     string `config:"DATABASE_USER" yaml:"user"`
	Password string `config:"DATABASE_PASSWORD" yaml:"password"`
	Host     string `config:"DATABASE_HOST" yaml:"host"`
	Port     int    `config:"DATABASE_PORT" yaml:"port"`
}
