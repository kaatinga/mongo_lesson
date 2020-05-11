package main

import (
	. "./models"
	"context"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"testing"
)

var (
	ctx = context.Background()
	err error
	client *mongo.Client
)

func init() {
	// Establishing connection to the database
	client, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatalln("Ошибка установки соединения")
	}
}

func TestHandlerData_Exist(t *testing.T) {

	type hd struct {
		Db                   *mongo.Database
		Data                 ViewData
		NoRender             bool
		FormID               string
		FormValue            string
		WhereToRedirect      string
		AdditionalRedirectID string
		MainAction           string
		Ctx                  context.Context
	}

	type args struct {
		id primitive.ObjectID
	}
	var ObjectID primitive.ObjectID
	ObjectID, err = primitive.ObjectIDFromHex("5ead1eaef50529bf0d6e4b39")
	if err != nil {
		t.Errorf("Couldn't get ObjectID from Hex")
	}

	tests := []struct {
		name    string
		fields  hd
		args    args
		want    bool
		wantErr bool
	}{
		{ "true", hd{}, args{ObjectID}, true, false },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hd := &HandlerData{
				Db:                   tt.fields.Db,
				Data:                 tt.fields.Data,
				NoRender:             tt.fields.NoRender,
				FormID:               tt.fields.FormID,
				FormValue:            tt.fields.FormValue,
				WhereToRedirect:      tt.fields.WhereToRedirect,
				AdditionalRedirectID: tt.fields.AdditionalRedirectID,
				MainAction:           tt.fields.MainAction,
				Ctx:                  tt.fields.Ctx,
			}

			hd.Db = client.Database("blog")
			got, err := hd.Exist(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Exist() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Exist() got = %v, want %v", got, tt.want)
			}
		})
	}
}
