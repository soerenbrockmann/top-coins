package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/streadway/amqp"
)

type Ranking struct {
	Ranking []CoinInfo `json:"Data"`
}

type CoinInfo struct {
	CoinInfo Name `json:"CoinInfo"`
}

type Name struct {
	Name string `json:"Name"`
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func getRanking(limit int) []byte {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://min-api.cryptocompare.com/data/top/mktcapfull", nil)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	q := url.Values{}
	q.Add("limit", strconv.Itoa(limit))
	q.Add("tsym", "USD")

	req.Header.Set("Accepts", "application/json")
	req.Header.Add("X-CMC_PRO_API_KEY", "<YOUR_API_KEY_HERE")
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request to server")
		os.Exit(1)
	}
	respBody, _ := ioutil.ReadAll(resp.Body)
	return respBody
}

func mapResponse(ranking []byte) []byte {
	var rankingData Ranking
	json.Unmarshal(ranking, &rankingData)
	var symbols []string
	for i := 0; i < len(rankingData.Ranking); i++ {
		symbols = append(symbols, rankingData.Ranking[i].CoinInfo.Name)
	}
	byteSymbols, _ := json.Marshal(symbols)
	return byteSymbols
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"ranking_queue",
		false,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare a queue")

	err = ch.Qos(
		1,
		0,
		false,
	)
	failOnError(err, "Failed to set QoS")

	msgs, err := ch.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			limit, err := strconv.Atoi(string(d.Body))
			failOnError(err, "Failed to convert body to integer")
			response := getRanking(limit)
			mappedResponse := mapResponse(response)
			err = ch.Publish(
				"",
				d.ReplyTo,
				false,
				false,
				amqp.Publishing{
					ContentType:   "text/plain",
					CorrelationId: d.CorrelationId,
					Body:          mappedResponse,
				})
			failOnError(err, "Failed to publish a message")

			d.Ack(false)
		}
	}()

	log.Printf(" [*] Awaiting RPC requests")
	<-forever
}
