package main

import (
	"context"
	"embed"
	"net/http"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/ugorji/go/codec"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/text/language"
)

//go:embed locales/*.toml
var localeFS embed.FS

type Configuration struct {
	MongoURL    string
	MongoDBName string

	ListenAddr string
}

type app struct {
	config      *Configuration
	codecHandle *codec.MsgpackHandle
	i18nBundle  *i18n.Bundle

	conn      *mongo.Client
	database  *mongo.Database
	users     *mongo.Collection
	feeds     *mongo.Collection
	seenItems *mongo.Collection
}

func main() {
	a := new(app)

	{
		cfgPath := os.Getenv("CONFIG_PATH")
		if len(cfgPath) == 0 {
			cfgPath = "config.toml"
		}
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			panic(err)
		}
		c := new(Configuration)
		toml.Unmarshal(data, c)
		a.config = c
	}

	{
		cl, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(a.config.MongoURL))
		if err != nil {
			panic(err)
		}
		err = cl.Ping(context.TODO(), nil)
		if err != nil {
			panic(err)
		}
		a.database = cl.Database(a.config.MongoDBName)
		a.users = a.database.Collection("users")
		a.feeds = a.database.Collection("feeds")
		a.seenItems = a.database.Collection("seen_items")
	}

	{
		msgpHandle := new(codec.MsgpackHandle)
		msgpHandle.WriterBufferSize = 8192
		msgpHandle.ReaderBufferSize = 8192
		a.codecHandle = msgpHandle
	}

	{
		bundle := i18n.NewBundle(language.BritishEnglish)
		bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
		_, _ = bundle.LoadMessageFileFS(localeFS, "translations/en.toml")
		_, _ = bundle.LoadMessageFileFS(localeFS, "translations/bn.toml")
		a.i18nBundle = bundle
	}

	http.HandleFunc("/", a.handler)
	http.ListenAndServe(a.config.ListenAddr, nil)
}
