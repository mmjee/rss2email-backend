package main

import (
	"context"
	"embed"
	"log"
	"net/http"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/caddyserver/certmagic"
	feed "github.com/mmcdole/gofeed"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/ugorji/go/codec"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/text/language"

	smtp "github.com/xhit/go-simple-mail/v2"
)

//go:embed locales/*.toml
var localeFS embed.FS

type Configuration struct {
	MongoURL    string
	MongoDBName string

	ListenAddr string
	BaseURL    string

	LetsEncrypt struct {
		Enable  bool
		Email   string
		Domains []string
	}

	EmailConfig struct {
		Security           smtp.Encryption
		AuthenticationType smtp.AuthType
		Port               uint
		FromAddr           string
		Hostname           string
		Username           string
		Password           string
	}
}

type app struct {
	config      *Configuration
	codecHandle *codec.MsgpackHandle
	i18nBundle  *i18n.Bundle
	emailClient *smtp.SMTPServer
	feedParser  *feed.Parser

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
		msgpHandle.WriteExt = true
		a.codecHandle = msgpHandle
	}

	{
		bundle := i18n.NewBundle(language.BritishEnglish)
		bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
		_, _ = bundle.LoadMessageFileFS(localeFS, "locales/en.toml")
		_, _ = bundle.LoadMessageFileFS(localeFS, "locales/bn.toml")
		a.i18nBundle = bundle
	}

	{
		srv := smtp.NewSMTPClient()
		srv.Authentication = a.config.EmailConfig.AuthenticationType
		srv.Encryption = a.config.EmailConfig.Security
		srv.Host = a.config.EmailConfig.Hostname
		srv.Port = int(a.config.EmailConfig.Port)
		srv.Username = a.config.EmailConfig.Username
		srv.Password = a.config.EmailConfig.Password
		a.emailClient = srv
	}

	{
		a.feedParser = feed.NewParser()
	}

	{
		err := a.EnsureIndexes()
		if err != nil {
			panic(err)
		}
	}
	go a.notificationLoop()

	log.Println("All initialized, listening.")
	http.HandleFunc("/", a.handler)
	if a.config.LetsEncrypt.Enable {
		certmagic.DefaultACME.Agreed = true
		certmagic.DefaultACME.Email = a.config.LetsEncrypt.Email
		certmagic.DefaultACME.CA = certmagic.LetsEncryptProductionCA
		err := certmagic.HTTPS(a.config.LetsEncrypt.Domains, http.DefaultServeMux)
		if err != nil {
			panic(err)
		}
	} else {
		err := http.ListenAndServe(a.config.ListenAddr, nil)
		if err != nil {
			panic(err)
		}
	}
}
