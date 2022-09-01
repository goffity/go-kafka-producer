package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/migrate"
	"go.uber.org/zap"
	"net/http"
	"os"
	"strings"
	"time"
)

func mapEnv() {
	for _, s := range os.Environ() {
		err := viper.BindEnv(strings.Split(s, "=")...)
		if err != nil {
			os.Exit(1)
		}
	}

	viper.AddConfigPath(".")
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		zap.S().Warnf("Not found config file: %s \n", err)
	}
}

func main() {
	mapEnv()

	migrate.NewMigrations()

	var logger *zap.Logger
	logger, _ = zap.NewDevelopment()
	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {

		}
	}(logger)

	zap.ReplaceGlobals(logger)

	zap.S().Infof("viper.GetString(\"KAFKA_BOOTSTRAP_SERVERS\"): %s", viper.GetString("KAFKA_BOOTSTRAP_SERVERS"))

	e := echo.New()
	e.GET("/", producer)

	e.Logger.Fatal(e.Start(":1323"))
}

func databaseConnection() *bun.DB {
	sqlDB, err := sql.Open("mysql", "root:rootPWD@tcp(DB:3306)/message_broker")
	if err != nil {
		panic(err)
	}
	return bun.NewDB(sqlDB, mysqldialect.New())
}

func producer(ctx echo.Context) error {
	viper.AutomaticEnv()

	conn := databaseConnection()

	hostname, _ := os.Hostname()
	produce, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":  viper.GetString("KAFKA_BOOTSTRAP_SERVERS"),
		"client.id":          hostname,
		"security.protocol":  viper.GetString("KAFKA_SECURITY_PROTOCOL"),
		"sasl.mechanisms":    viper.GetString("KAFKA_SASL_MECHANISMS"),
		"sasl.username":      viper.GetString("KAFKA_SASL_USERNAME"),
		"sasl.password":      viper.GetString("KAFKA_SASL_PASSWORD"),
		"request.timeout.ms": viper.GetString("KAFKA_REQUEST_TIMEOUT_MS"),
		"acks":               "all"},
	)
	if err != nil {
		zap.S().Errorf("Failed to create producer: %s\n", err)
	}
	defer produce.Close()

	topic := "test-topic"
	message := fmt.Sprintf("message_%s", time.Now().String())

	go func() {
		status := ""
		details := ""
		for e := range produce.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					zap.S().Infof("Failed to deliver message: %v\n", ev.TopicPartition)
					status = "failed"
					details = ev.TopicPartition.Error.Error()
				} else {
					status = "sent"
				}
			case kafka.Error:
				zap.S().Errorf("ERROR: %s", ev.Error())
				status = "failed"
				details = ev.Error()
			default:
				zap.S().Infof("default %T", ev)
			}
		}

		_, err = conn.NewInsert().Model(&Message{Message: message, Status: status, Details: details}).Exec(context.Background())
		if err != nil {
			panic(err)
		}
	}()

	err = produce.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          []byte(message),
	}, nil)
	if err != nil {
		zap.S().Errorf("fail to produce message: %s to topic: %s", message, topic)
		return err
	}
	_, err = conn.NewInsert().Model(&Message{Message: message, Status: "sending"}).Exec(context.Background())
	if err != nil {
		panic(err)
	}

	produce.Flush(15 * 1000)

	return ctx.JSON(http.StatusOK, `OK`)
}
