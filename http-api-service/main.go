package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"

	"github.com/streadway/amqp"
)

type Prices struct {
	Items []Price
}

type Price struct {
	Symbol string
	Price  float32
}

type Ranking struct {
	Items []Rank
}

type Rank struct {
	Rank   int
	Symbol string
	Price  float32
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func randomString(l int) string {
	bytes := make([]byte, l)
	for i := 0; i < l; i++ {
		bytes[i] = byte(randInt(65, 90))
	}
	return string(bytes)
}

func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}

func handleRPC(limit int, queue string) (res []byte, err error) {
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"",
		false,
		false,
		true,
		false,
		nil,
	)
	failOnError(err, "Failed to declare a queue")

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to register a consumer")

	corrId := randomString(32)

	err = ch.Publish(
		"",
		queue,
		false,
		false,
		amqp.Publishing{
			ContentType:   "text/plain",
			CorrelationId: corrId,
			ReplyTo:       q.Name,
			Body:          []byte(strconv.Itoa(limit)),
		})
	failOnError(err, "Failed to publish a message")

	for d := range msgs {
		if corrId == d.CorrelationId {
			res = d.Body
			failOnError(err, "Failed to convert body to integer")
			break
		}
	}

	return
}

func (ranking *Ranking) AddItem(rank Rank) {
	ranking.Items = append(ranking.Items, rank)
}

func mapData(ranking []byte, prices []byte) Ranking {
	var parsedPrices Prices
	json.Unmarshal(prices, &parsedPrices)
	var parsedRanking []string
	json.Unmarshal(ranking, &parsedRanking)
	rankingData := Ranking{}
	for i := 0; i < len(parsedRanking); i++ {
		var price float32
		for j := range parsedPrices.Items {
			if parsedPrices.Items[j].Symbol == parsedRanking[i] {
				price = parsedPrices.Items[j].Price
				break
			}
		}

		rank := Rank{
			Rank:   i + 1,
			Symbol: parsedRanking[i],
			Price:  price,
		}
		rankingData.AddItem(rank)

	}
	return rankingData
}

func getLimit(url *url.URL) int {
	params, ok := url.Query()["limit"]
	limit := 10
	if ok {
		n, err := strconv.Atoi(params[0])
		failOnError(err, "Failed to convert arg to integer")
		limit = n
		if n > 100 {
			limit = 100
		} else {
			limit = n
		}
	}
	return limit
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		limit := getLimit(r.URL)
		ranking, err := handleRPC(limit, "ranking_queue")
		failOnError(err, "Failed to handle RPC request")
		// Cannot fetch prices by Symbol. Hence I need to fetch all price data.
		prices, err := handleRPC(5000, "pricing_queue")
		failOnError(err, "Failed to handle RPC request")
		rankingResult := mapData(ranking, prices)

		json.NewEncoder(w).Encode(rankingResult)
	})
	log.Fatal(http.ListenAndServe(":8081", nil))

}
