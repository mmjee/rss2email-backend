package main

import (
	"context"
	"encoding/hex"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"git.maharshi.ninja/root/rss2email/structures"
	"github.com/mmcdole/gofeed"
	smtp "github.com/xhit/go-simple-mail/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/sync/errgroup"
)

var (
	tickerTime           = 10 * time.Minute
	tickerTimeHalf       = tickerTime / 2
	batchSize      int64 = 1000
)

type FeedWithUser struct {
	*structures.Feed `bson:",inline"`
	OwnerList        []*structures.User `bson:"owner_list"`
}

func (a *app) sendEmailForItem(feed *structures.Feed, user *structures.User, item *gofeed.Item) error {
	bldr := new(strings.Builder)
	// Not bothering to use the localizer or a template (because templates are painful) and because the locale isn't known, so we have to use English, unfortunately
	bldr.WriteString("New post on ")
	bldr.WriteString(feed.Name)
	bldr.WriteString(": ")
	bldr.WriteString(item.Title)

	subject := bldr.String()

	bldr.WriteString("\nURL: ")
	bldr.WriteString(item.Link)
	bldr.WriteRune('\n')
	if len(item.Links) != 0 {
		bldr.WriteString("Additional Links:\n")
		for num, link := range item.Links {
			bldr.WriteString(strconv.Itoa(num + 1))
			bldr.WriteString(". ")
			bldr.WriteString(link)
			bldr.WriteRune('\n')
		}
	}
	bldr.WriteString("\n\n")
	bldr.WriteString(item.Content)
	body := bldr.String()

	msg := smtp.NewMSG()
	msg.SetFrom(a.config.EmailConfig.FromAddr)
	msg.SetSubject(subject)
	msg.AddTo(user.Email)
	msg.SetBody(smtp.TextPlain, body)

	conn, err := a.emailClient.Connect()
	if err != nil {
		return err
	}

	err = msg.Send(conn)
	if err != nil {
		return err
	}

	return nil
}

func (a *app) onNotificationTick(tym time.Time) {
	totalCount, err := a.feeds.CountDocuments(context.TODO(), bson.M{})
	if err != nil {
		panic(err)
	}
	timeUnix := tym.UnixNano()
	batches := int64(math.Ceil(float64(totalCount) / float64(batchSize)))
	log.Printf("Processing %d batches\n", batches)

	// outer:
	for counter := int64(0); counter != batches; counter++ {
		var feedList []FeedWithUser
		log.Printf("Processing %d\n", counter)

		crsr, err := a.feeds.Aggregate(context.TODO(), bson.A{
			bson.M{
				"$limit": batchSize,
			},
			bson.M{
				"$skip": counter * batchSize,
			},
			bson.M{
				"$lookup": bson.M{
					"from":         "users",
					"localField":   "owner_id",
					"foreignField": "_id",
					"as":           "owner_list",
				},
			},
		}, options.Aggregate().SetAllowDiskUse(true).SetBatchSize(1000))
		if err != nil {
			panic(err)
		}
		err = crsr.All(context.TODO(), &feedList)
		if err != nil {
			panic(err)
		}

		grp := new(errgroup.Group)

		for _, _feedDoc := range feedList {
			feedDoc := _feedDoc

			if len(feedDoc.OwnerList) == 0 {
				log.Printf("WARNING! Orphaned feed: %s\n", hex.EncodeToString(feedDoc.ID[:]))
				continue
			}

			diff := time.Duration(timeUnix) % feedDoc.Frequency
			// If diff is greater than tickerTime or less than -tickerTime, skip because it's not relevant
			if diff >= tickerTime || diff <= -tickerTime {
				continue
			}

			if !feedDoc.OwnerList[0].EmailVerified {
				continue
			}

			grp.Go(func() error {
				feed, err := a.feedParser.ParseURL(feedDoc.URL)
				if err != nil {
					log.Printf("Couldn't fetch %s\n", hex.EncodeToString(feedDoc.ID[:]))
					return nil // doesn't need to interrupt the fetching
				}

				for _, item := range feed.Items {
					// Do NOT report items created before the feed creation unless NotifyOldItems is set to true
					if feedDoc.CreatedAt.After(*item.PublishedParsed) && a.config.NotifyOldItems == false {
						continue
					}

					exists, err := a.seenItems.CountDocuments(context.TODO(), bson.M{
						"feed_id": feedDoc.ID,
						"guid":    item.GUID,
					})
					if err != nil {
						log.Printf("Tried to find %s (%s), failed with %s\n", item.GUID, hex.EncodeToString(feedDoc.ID[:]), err.Error())
						continue
					}
					if exists != 0 {
						continue
					}
					err = a.sendEmailForItem(feedDoc.Feed, feedDoc.OwnerList[0], item)
					if err != nil {
						log.Printf("Failed while sending email for %s (%s), failed with %s\n", item.GUID, hex.EncodeToString(feedDoc.ID[:]), err.Error())
						continue
					}
					_, err = a.seenItems.InsertOne(context.TODO(), structures.SeenItem{
						ID:        primitive.NewObjectID(),
						FeedID:    feedDoc.ID,
						GUID:      item.GUID,
						Timestamp: time.Now(),
					})
					if err != nil {
						log.Printf("Failed while inserting seen item for %s (%s), failed with %s\n", item.GUID, hex.EncodeToString(feedDoc.ID[:]), err.Error())
						continue
					}
				}

				_, err = a.feeds.UpdateByID(context.TODO(), feedDoc.ID, bson.M{
					"$set": bson.M{
						"last_fetched": time.Now(),
					},
				})

				if err != nil {
					log.Printf("Failed while updating last_fetched for %s, failed with %s\n", hex.EncodeToString(feedDoc.ID[:]), err.Error())

				}

				return nil
			})
		}

		err = grp.Wait()
		if err != nil {
			panic(err)
		}

		err = crsr.Close(context.TODO())
		if err != nil {
			log.Printf("Failed while closing the cursor: %s\n", err.Error())
			return
		}
	}
}

func (a *app) notificationLoop() {
	go a.onNotificationTick(time.Now())
	ticker := time.NewTicker(tickerTime)
	for t := range ticker.C {
		a.onNotificationTick(t)
	}
}
