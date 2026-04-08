package db

import "go.mongodb.org/mongo-driver/bson/primitive"

func ToObjectID(id string) primitive.ObjectID {
	objId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		println(err.Error())
	}
	return objId
}
