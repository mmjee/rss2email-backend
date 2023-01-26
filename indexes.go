package main

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (a *app) EnsureIndexes() error {
	{
		seenItemsView := a.seenItems.Indexes()
		_, err := seenItemsView.CreateMany(context.TODO(), []mongo.IndexModel{
			{
				Keys: bson.D{
					{Key: "feed_id", Value: -1},
					{Key: "guid", Value: -1},
				},
				Options: options.Index().SetName("already_notified_lookup").SetUnique(true),
			},
		})
		if err != nil {
			return err
		}
	}
	{
		usersView := a.users.Indexes()
		_, err := usersView.CreateMany(context.TODO(), []mongo.IndexModel{
			{
				Keys:    bson.D{{Key: "email", Value: -1}},
				Options: options.Index().SetName("preexisting_email_lookup").SetUnique(true),
			},
			{
				Keys:    bson.D{{Key: "addr", Value: -1}},
				Options: options.Index().SetName("address_based_lookup").SetUnique(true),
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}
