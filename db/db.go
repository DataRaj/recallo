package db

import "go.mongodb.org/mongo-driver/v2/bson"

func ToObjectID(id string) bson.ObjectID {
	objId, err := bson.ObjectIDFromHex(id)
	if err != nil {
		println(err.Error())
	}
	return objId
}
