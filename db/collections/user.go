package collections

import (
	"context"
	"gotel/types"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const DBNAME = "gotel-reservation"
const userColl = "user"

type UserStore interface {
	GetUserByID(context.Context, string) (*types.User, error)
}

type MongoUserStore struct {
	client *mongo.Client
	coll   *mongo.Collection
}

func NewMongoUserStore(client *mongo.Client) *MongoUserStore {
	return &MongoUserStore{
		client: client,
		coll:   client.Database(DBNAME).Collection(userColl),
	}
}

// this is for creating an mongo based function to fetch the user
func (u *MongoUserStore) GetUserByID(ctx context.Context, id string) (*types.User, error) {
	oid, _ := bson.ObjectIDFromHex(id)
	var user types.User
	if err := u.coll.FindOne(ctx, bson.M{"_id": (oid)}).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}
