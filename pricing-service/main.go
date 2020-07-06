package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/streadway/amqp"
)

type Prices struct {
	Prices []Price `json:"data"`
}

type Price struct {
	Symbol string `json:"symbol"`
	Quote  struct {
		USD struct {
			Price float32 `json:"price"`
		} `json:"USD"`
	} `json:"quote"`
}

type MappedPrices struct {
	Items []MappedPrice
}

type MappedPrice struct {
	Symbol string
	Price  float32
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func getPrices() []byte {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://pro-api.coinmarketcap.com/v1/cryptocurrency/listings/latest", nil)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	q := url.Values{}
	q.Add("start", "1")
	q.Add("limit", "5000")
	q.Add("convert", "USD")

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

func (mappedPrices *MappedPrices) AddItem(mappedPrice MappedPrice) {
	mappedPrices.Items = append(mappedPrices.Items, mappedPrice)
}

func mapPrices(prices []byte) []byte {
	var parsedPrices Prices
	json.Unmarshal(prices, &parsedPrices)
	mappedPrices := MappedPrices{}

	for i := 0; i < len(parsedPrices.Prices); i++ {
		price := MappedPrice{
			Symbol: parsedPrices.Prices[i].Symbol,
			Price:  parsedPrices.Prices[i].Quote.USD.Price,
		}
		mappedPrices.AddItem(price)
	}
	bytePrices, _ := json.Marshal(mappedPrices)
	return bytePrices
}

func main() {

	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"pricing_queue",
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
			response := getPrices()
			mappedResponse := mapPrices(response)
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
