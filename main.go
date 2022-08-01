package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/SlyMarbo/rss"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/disgo/webhook"
	"github.com/disgoorg/snowflake/v2"
	"github.com/madflojo/tasks"
	"go.uber.org/zap"
)

func main() {

	// create zap logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// use sugar version
	sugar := logger.Sugar()
	defer sugar.Sync()

	sugar.Info("Started")

	feed, err := rss.Fetch(getEnv("RSS_Feed", "https://example.com/feed/"))

	if err != nil {
		sugar.Warnw("An error occurred with the feed reader", "error", err)
	}

	scheduler := tasks.New()
	defer scheduler.Stop()

	defer sugar.Info("Shutting down...")

	client := webhook.New(snowflake.ID(stringToNum(getEnv("Webhook_ID", "123"))), getEnv("Webhook_Token", "TOKEN"))
	defer client.Close(context.TODO())

	// Add a task
	_, err = scheduler.Add(&tasks.Task{
		Interval: time.Duration(10 * time.Minute),
		TaskFunc: func() error {

			err := feed.Update()

			if err != nil {
				sugar.Warnw("An error occurred with the feed reader", "error", err)
				return err
			}

			sugar.Info("Updated feed")

			processFeed(feed, sugar, client)

			return nil
		},
	})
	if err != nil {
		// Do Stuff
		sugar.Panicw("Error occurred defining job", "error", err)
	}

	processFeed(feed, sugar, client)

	// Wait for the application to be terminated
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt)
	<-shutdown
}

func processFeed(feed *rss.Feed, sugar *zap.SugaredLogger, client webhook.Client) {
	sugar.Info("Processing feed")
	defer sugar.Info("Processed feed")

	for i := range feed.Items {
		product := feed.Items[i]

		country := getEnv("Country", "US")

		// check if day of year is before today
		previousDay := time.Now().YearDay() > product.Date.YearDay()

		// check if seen or on previous day
		if previousDay || product.Read {
			continue
		}

		for j := range product.Categories {
			if product.Categories[j] == country {
				notify(product, *sugar, client)
			}
		}
		product.Read = true
	}
}

func notify(product *rss.Item, sugar zap.SugaredLogger, client webhook.Client) {
	country := getEnv("Country", "US")

	sugar.Infof("Found %s in %s", product.Categories[2], country)

	// embed := discord.NewEmbedBuilder().SetTitle(product.Title).SetDescription(product.Summary).SetTimestamp(product.Date).SetURL(product.Link).AddField("Country", product.Categories[1], true).AddField("Product", product.Categories[2], true)

	// client.CreateEmbeds()

	if _, err := client.CreateMessage(discord.NewWebhookMessageCreateBuilder().SetContentf("**%s**\n%s\n%s", product.Title, product.Summary, product.Link).Build(),
		// delay each request by 2 seconds
		rest.WithDelay(2*time.Second),
	); err != nil {
		sugar.Errorw("Error with sending Discord webhook", "error", err)
	}
}

func getEnv(key string, defaultValue string) string {
	val, ok := os.LookupEnv(key)

	if !ok {
		return defaultValue
	} else {
		return val
	}
}

func stringToNum(str string) int {
	n, _ := strconv.Atoi(str)
	return n
}
