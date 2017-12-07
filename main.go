package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"gotrading/core"
	"gotrading/graph"
	"gotrading/services"
	"gotrading/strategies"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges/bittrex"
	"github.com/thrasher-/gocryptotrader/exchanges/kraken"
	"github.com/thrasher-/gocryptotrader/exchanges/liqui"

	"flag"
)

var (
	uri          = flag.String("uri", "amqp://developer:xLae4pzT@hc-amqp.dev:5672/hc", "AMQP URI")
	exchangeName = flag.String("exchange", "arbitrage.fanout", "Durable AMQP exchange name")
	exchangeType = flag.String("exchange-type", "fanout", "Exchange type - direct|fanout|topic|x-custom")
	routingKey   = flag.String("key", "test-key", "AMQP routing key")
	body         = flag.String("body", "foobar", "Body of message")
	reliable     = flag.Bool("reliable", true, "Wait for the publisher confirmation before exiting")
)

func init() {
	flag.Parse()
}

func main() {

	cfg := &config.Cfg
	err := cfg.LoadConfig("config.dat")
	if err != nil {
		log.Fatal(err)
	}

	interrupt := make(chan os.Signal, 1)

	liquiEngine := new(liqui.Liqui)
	krakenEngine := new(kraken.Kraken)
	bittrexEngine := new(bittrex.Bittrex)
	// gdaxEngine := new(gdax.GDAX)
	// poloniexEngine := new(poloniex.Poloniex)

	liqui := services.LoadExchange(cfg, "Liqui", liquiEngine)
	kraken := services.LoadExchange(cfg, "Kraken", krakenEngine)
	bittrex := services.LoadExchange(cfg, "Bittrex", bittrexEngine)
	// poloniex := services.LoadExchange(cfg, "Poloniex", poloniexEngine)
	// gdax := services.LoadExchange(cfg, "GDAX", gdaxEngine)
	bittrex.IsCurrencyPairNormalized = false
	kraken.IsCurrencyPairNormalized = true

	// exchanges := []core.Exchange{kraken, liqui, gdax, bittrex}
	exchanges := []core.Exchange{kraken, liqui}
	// exchanges := []core.Exchange{liqui}

	mashup := core.ExchangeMashup{}
	mashup.Init(exchanges)

	from := core.Currency("BTC")
	to := from
	depth := 3
	treeOfPossibles, _, _, _ := graph.PathFinder(mashup, from, to, depth)

	fmt.Println(treeOfPossibles.Description())

	arbitrage := strategies.Arbitrage{}

	delayBetweenReqs := make(map[string]time.Duration, len(exchanges))
	delayBetweenReqs["Kraken"] = time.Duration(100)
	delayBetweenReqs["Liqui"] = time.Duration(100)
	delayBetweenReqs["Bittrex"] = time.Duration(100)

	// Rabbit
	// conn, err := amqp.Dial("amqp://developer:xLae4pzT@hc-amqp.dev:5672/hc")
	// failOnError(err, "Failed to connect to RabbitMQ")
	// defer conn.Close()

	// ch, err := conn.Channel()
	// failOnError(err, "Failed to open a channel")
	// defer ch.Close()

	// err = ch.ExchangeDeclare(
	// 	"arbitrage.routing", // name
	// 	"topic",             // type
	// 	true,                // durable
	// 	false,               // auto-deleted
	// 	false,               // internal
	// 	false,               // no-wait
	// 	nil,                 // arguments
	// )
	// failOnError(err, "Failed to declare an exchange")

	for {
		treeOfPossibles.DepthTraversing(func(vertices []*graph.Vertice) {
			services.FetchVertices(vertices, func(path graph.Path) {
				chains := arbitrage.Run([]graph.Path{path})
				rows := make([][]string, 0)
				for _, chain := range chains {
					if chain.Performance == 0 {
						continue
					}
					ordersCount := len(chain.Path.Nodes)
					row := make([]string, ordersCount+5)
					for j, node := range chain.Path.Nodes {
						row[j] = node.Description()
					}

					row[ordersCount] = strconv.FormatFloat(chain.Performance, 'f', 6, 64)
					row[ordersCount+1] = strconv.FormatFloat(chain.VolumeToEngage, 'f', 6, 64)
					row[ordersCount+2] = strconv.FormatFloat(chain.VolumeToEngage*chain.Performance, 'f', 6, 64)
					row[ordersCount+3] = strconv.FormatFloat(chain.VolumeOut, 'f', 6, 64)

					t := time.Now()
					row[ordersCount+4] = t.Format("2006-01-02 15:04:05")
					rows = append(rows, row)
					fmt.Println(strings.Join(row[:], ","))

					// marshal, _ := json.Marshal(chain)
					// err = ch.Publish(
					// 	"arbitrage.routing", // exchange
					// 	"usd.btc",           // routing key
					// 	false,               // mandatory
					// 	false,               // immediate
					// 	amqp.Publishing{
					// 		ContentType: "text/plain",
					// 		Body:        []byte(marshal),
					// 	})
				}
				if len(rows) > 0 {
					// table.AppendBulk(rows)
					// table.Render()
				}

			})
			time.Sleep(2000 * time.Millisecond)
		})
	}

	<-interrupt
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}
