package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/antchfx/htmlquery"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/net/proxy"
)

// Show struct contains show id
type Show struct {
	ID string
}

// SensitiveData struct contains all secret keys for proxy and Telegram
type SensitiveData struct {
	BotToken        string `json:"token"`
	User            string `json:"user"`
	Pass            string `json:"pass"`
	Port            int    `json:"port"`
	Host            string `json:"host"`
	MongoURI        string `json:"mongostring"`
	MongoDB         string `json:"mongodb"`
	MongoCollection string `json:"mongocollection"`
}

func createMongoConnection(mongoURI string) *mongo.Client {
	clientOptions := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}

	return client
}

func createProxyHTTPClient(user string, pass string, host string, port int) *http.Client {
	auth := proxy.Auth{
		User:     user,
		Password: pass,
	}

	dialSocksProxy, err := proxy.SOCKS5("tcp", fmt.Sprintf("%s:%d", host, port), &auth, proxy.Direct)
	if err != nil {
		fmt.Println("Error connecting to proxy:", err)
	}
	tr := &http.Transport{Dial: dialSocksProxy.Dial}

	// Create client
	myClient := &http.Client{
		Transport: tr,
	}

	return myClient
}

func botWork(client *http.Client, token string) *tgbotapi.BotAPI {
	bot, err := tgbotapi.NewBotAPIWithClient(token, client)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	return bot
}

func getSensitiveData(jsonPath string) SensitiveData {
	jsonFile, err := os.Open(jsonPath)
	if err != nil {
		log.Fatal(err)
	}

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var data SensitiveData

	json.Unmarshal(byteValue, &data)

	return data
}

func check(url string, id int64, bot *tgbotapi.BotAPI, collection *mongo.Collection) {
	doc, err := htmlquery.LoadURL(url)
	if err != nil {
		panic(err)
	}

	for _, n := range htmlquery.Find(doc, "//a[@class='order']") {
		a := htmlquery.FindOne(n, "//a[@class='order']")

		showID := htmlquery.SelectAttr(a, "data-id")
		filter := bson.D{{"id", showID}}

		var result Show
		err = collection.FindOne(context.TODO(), filter).Decode(&result)
		if err != nil {
			text := fmt.Sprintf("New show! Maybe it is Kuji, check faster: https://standupstore.ru")
			msg := tgbotapi.NewMessage(id, text)
			bot.Send(msg)

			show := Show{showID}
			_, err := collection.InsertOne(context.TODO(), show)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func main() {

	var id int64 = -388648621

	var dataPath string

	if _, err := os.Stat("local_sensitive_data.json"); err == nil {
		dataPath = "local_sensitive_data.json"
	} else if os.IsNotExist(err) {
		dataPath = "sensitive_data.json"
	}

	data := getSensitiveData(dataPath)

	customHTTPClient := createProxyHTTPClient(data.User, data.Pass, data.Host, data.Port)
	bot := botWork(customHTTPClient, data.BotToken)

	mongoClient := createMongoConnection(data.MongoURI)
	collection := mongoClient.Database(data.MongoDB).Collection(data.MongoCollection)

	for {

		for i := 1; i < 7; i++ {
			url := fmt.Sprintf("https://standupstore.ru/page/%d/", i)
			check(url, id, bot, collection)
		}

		time.Sleep(time.Second * 55)
	}

}
