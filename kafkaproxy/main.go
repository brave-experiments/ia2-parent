package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"

	msg "github.com/brave-experiments/ia2/message"
	"github.com/segmentio/kafka-go"
)

const (
	defaultListenAddr = "127.0.0.1:8081"
	defaultKafkaKey   = "/etc/kafka/secrets/key"
	defaultKafkaCert  = "/etc/kafka/secrets/certificate"

	kafkaTestTopic = "antifraud_client_addrs_events.testing.repsys.upstream"
	envKafkaBroker = "KAFKA_BROKERS"

	typeNumReqs = iota
	typeNumGoodFwds
	typeNumBadFwds
)

var l = log.New(os.Stderr, "kafkaproxy: ", log.Ldate|log.Ltime|log.LUTC|log.Lshortfile)

// kafkaProxy represents a proxy that takes as input HTTP requests, turns them
// into Kafka messages, and sends them to a broker.  Think of the proxy as an
// HTTP-to-Kafka bridge.
type kafkaProxy struct {
	writer *kafka.Writer
	stats  *statistics
	//addrs  *addresses
	addrs msg.WalletsByKeyID
}

// addresses represents a mapping from a hashed key to a slice of addresses
// that were anonymized using the hashed key.
// type addresses struct {
// 	Addrs map[string][]string `json:"addrs"`
// }

// statistics represents simple statistics of the Kafka proxy.
type statistics struct {
	sync.Mutex
	numReqs     int
	numGoodFwds int
	numBadFwds  int
}

// inc increments the given statistic.
func (s *statistics) inc(statType int) {
	s.Lock()
	defer s.Unlock()

	switch statType {
	case typeNumReqs:
		s.numReqs++
	case typeNumGoodFwds:
		s.numGoodFwds++
	case typeNumBadFwds:
		s.numBadFwds++
	}
}

// get returns the given statistic.
func (s *statistics) get(statType int) int {
	s.Lock()
	defer s.Unlock()

	switch statType {
	case typeNumReqs:
		return s.numReqs
	case typeNumGoodFwds:
		return s.numGoodFwds
	case typeNumBadFwds:
		return s.numBadFwds
	}
	return 0
}

// forwards forwards the currently cached addresses to the Kafka broker.  If
// anything goes wrong, the function returns an error.
func (p *kafkaProxy) forward() error {
	jsonStr, err := json.Marshal(p.addrs)
	if err != nil {
		return err
	}

	err = p.writer.WriteMessages(context.Background(),
		kafka.Message{
			Key:   nil,
			Value: []byte(jsonStr),
		},
	)
	if err != nil {
		return err
	}

	return nil
}

// getAddressesHandler returns an HTTP handler that answers /addresses
// requests, i.e., submissions of freshly anonymizes IP addresses.
func getAddressesHandler(p *kafkaProxy) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p.stats.inc(typeNumReqs)
		if r.Method != http.MethodPost {
			http.Error(w, "only POST requests are accepted", http.StatusBadRequest)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var addrs = make(msg.WalletsByKeyID)
		if err := addrs.UnmarshalJSON(body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := p.forward(); err != nil {
			http.Error(w, fmt.Sprintf("failed to forward addresses: %s", err),
				http.StatusInternalServerError)
			p.stats.inc(typeNumBadFwds)
			return
		}
		p.stats.inc(typeNumGoodFwds)
	}
}

// getStatusHandler returns an HTTP handler that answers /status requests,
// i.e., requests for the proxy's statistics.
func getStatusHandler(p *kafkaProxy) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "# of forward requests: %d\n"+
			"# of successful forwards: %d\n"+
			"# of failed forwards: %d\n",
			p.stats.get(typeNumReqs),
			p.stats.get(typeNumGoodFwds),
			p.stats.get(typeNumBadFwds))
	}
}

// newKafkaProxy creates a new Kafka proxy from the environment variables
// envKafkaBroker and envKafkaCert.
func newKafkaProxy(certFile, keyFile string) (*kafkaProxy, error) {
	p := &kafkaProxy{
		stats: new(statistics),
		addrs: make(msg.WalletsByKeyID),
	}

	kafkaBroker, exists := os.LookupEnv(envKafkaBroker)
	if !exists {
		return nil, fmt.Errorf("environment variable %q not set", envKafkaBroker)
	}
	if kafkaBroker == "" {
		return nil, fmt.Errorf("environment variable %q empty", envKafkaBroker)
	}
	l.Printf("Fetched Kafka broker %q from environment variable.", kafkaBroker)

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	l.Println("Loaded certificate and key file for Kafka.")

	p.writer = &kafka.Writer{
		Addr:  kafka.TCP(kafkaBroker),
		Topic: kafkaTestTopic,
		Transport: &kafka.Transport{
			TLS: &tls.Config{Certificates: []tls.Certificate{cert}},
		},
	}

	return p, nil
}

func main() {
	var err error
	var proxy *kafkaProxy

	var cert = flag.String("cert", "", "Path to Kafka certificate.")
	var key = flag.String("key", "", "Path to Kafka key.")
	var listen = flag.String("listen", "", "Address to listen on.")
	flag.Parse()

	if *cert == "" || *key == "" {
		*cert = defaultKafkaCert
		*key = defaultKafkaKey
		l.Printf("Arguments -cert and -key not set.  Using %s and %s.", *cert, *key)
	}
	if *listen == "" {
		*listen = defaultListenAddr
		l.Printf("Argument -listen not set.  Using %s.", *listen)
	}

	if proxy, err = newKafkaProxy(*cert, *key); err != nil {
		l.Fatalf("Failed to create Kafka writer: %s", err)
	}

	http.HandleFunc("/addresses", getAddressesHandler(proxy))
	http.HandleFunc("/status", getStatusHandler(proxy))
	l.Println("Starting Kafka proxy.")
	l.Fatal(http.ListenAndServe(*listen, nil))
}
