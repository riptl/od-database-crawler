package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	ServerUrl string
	Token string
}


func main2() {
	var err error

	viper.SetConfigName("config.yml")
	viper.SetConfigType("yml")
	err = viper.ReadInConfig()
	if err != nil {
		logrus.Fatal(err)
	}
}
