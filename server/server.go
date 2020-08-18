package server

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"time"

	"github.com/jbltx/master-server/valve"

	"github.com/jbltx/master-server/config"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/bson"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var gameServersCollection *mongo.Collection
var challengesCollection *mongo.Collection

type MasterServer struct {
	cfg config.Config
}

func NewMasterServer(cfg config.Config) *MasterServer {
	return &MasterServer{
		cfg: cfg,
	}
}

type GameServer struct {
	ID                primitive.ObjectID `bson:"_id,omitempty"`
	EndpointID        int64              `bson:"endpointID,omitempty"`
	IP                string             `bson:"ip,omitempty"`
	Port              int32              `bson:"port,omitempty"`
	LastHeartbeatDate time.Time          `bson:"lastHeartbeatDate,omitempty"`
}

type Challenge struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	EndpointID int64              `bson:"endpointID,omitempty"`
	Value      int32              `bson:"value,omitempty"`
	UpdatedAt  time.Time          `bson:"updatedAt,omitempty"`
}

type ServerEndpoint struct {
	IP   net.IP
	Port uint16
}

var nullEndpoint ServerEndpoint = ServerEndpoint{
	IP:   net.ParseIP("0.0.0.0"),
	Port: 0,
}

func NewServerEndpoint(gameServer *GameServer) *ServerEndpoint {
	return &ServerEndpoint{
		IP:   net.ParseIP(gameServer.IP),
		Port: uint16(gameServer.Port),
	}
}

func (c *ServerEndpoint) String() string {
	p := strconv.Itoa(int(c.Port))
	return c.IP.String() + ":" + p
}

func (c *ServerEndpoint) Bytes() []byte {

	buffer := new(bytes.Buffer)
	buffer.WriteByte(c.IP[0])
	buffer.WriteByte(c.IP[1])
	buffer.WriteByte(c.IP[2])
	buffer.WriteByte(c.IP[3])
	binary.Write(buffer, binary.BigEndian, uint16(c.Port))

	return buffer.Bytes()
}

func (c *ServerEndpoint) Uint64() uint64 {
	arr := []byte{0x00, 0x00, byte(c.Port >> 8), byte(c.Port & 0xFF), c.IP[0], c.IP[1], c.IP[2], c.IP[0]}
	ret := binary.BigEndian.Uint64(arr)

	// log.Println("The endpoint " + c.String() + " has a hash value equals to " + strconv.Itoa(int(ret)))

	return ret
}

func handleServerListRequest(buffer []byte, endpoint *ServerEndpoint) []byte {
	response := new(bytes.Buffer)
	response.Write(valve.ServerListHeader)

	opts := options.Find().SetSort(bson.D{{"endpointID", 1}}).SetLimit(config.ServerListMaxCount)
	filter := bson.D{{"endpointID", bson.D{{"$gt", endpoint.Uint64()}}}}
	cursor, err := gameServersCollection.Find(context.TODO(), filter, opts)

	if err != nil {
		log.Fatal(err)
	}

	returnedServerCount := cursor.RemainingBatchLength()

	for cursor.Next(context.TODO()) {
		var gameServer GameServer
		if err = cursor.Decode(&gameServer); err != nil {
			log.Fatal(err)
		}
		endpoint := NewServerEndpoint(&gameServer)
		response.Write(endpoint.Bytes())
	}

	if returnedServerCount < int(config.ServerListMaxCount) {
		response.Write(nullEndpoint.Bytes())
	}

	return response.Bytes()
}

func handleJoinRequest(endpoint *ServerEndpoint) []byte {
	response := new(bytes.Buffer)
	response.Write(valve.ChallengeHeader)
	challengeNumber := int32(rand.Int())
	binary.Write(response, binary.BigEndian, challengeNumber)
	opts := options.Update().SetUpsert(true) // create a new document if not already here
	filter := bson.D{{"endpointID", endpoint.Uint64()}}
	update := bson.D{{"$set", bson.D{{"value", challengeNumber}, {"updatedAt", time.Now()}}}}
	res, err := challengesCollection.UpdateOne(context.TODO(), filter, update, opts)
	if err != nil {
		log.Fatal(err)
	}
	if res.UpsertedCount == 0 {
		log.Println("[INFO] JOIN - An endpoint has been updated in the challenge database (" + endpoint.String() + ")")
	} else {
		log.Println("[INFO] JOIN - A new endpoint has been added in the challenge database (" + endpoint.String() + ")")
	}
	return response.Bytes()
}

func handleQuitRequest(endpoint *ServerEndpoint) {
	res, err := gameServersCollection.DeleteOne(context.TODO(), bson.D{{"endpointID", endpoint.Uint64()}})
	if err != nil {
		log.Fatal(err)
	}
	if res.DeletedCount == 0 {
		log.Println("[WARN] QUIT - Received a request for an unknown endpoint (" + endpoint.String() + ")")
	} else {
		log.Println("[INFO] QUIT - An endpoint has been removed from the database (" + endpoint.String() + ")")
	}
}

func handleChallengeRequest(req []byte, endpoint *ServerEndpoint) {
	var challengeReq valve.ChallengeRequest
	err := valve.UnmarshallChallenge(req, &challengeReq)
	if err != nil {
		log.Fatal(err)
	}
	filter := bson.D{{"endpointID", endpoint.Uint64()}}
	var challengeEntry Challenge
	err = challengesCollection.FindOne(context.TODO(), filter).Decode(&challengeEntry)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// ? (jbltx) blacklist endpoint ?
			return
		}
		log.Fatal(err)
	}
	challengesCollection.DeleteOne(context.TODO(), filter)
	if challengeEntry.Value != challengeReq.ChallengeValue {
		// ? (jbltx) blacklist endpoint ?
	} else {
		// todo (jbltx) For now we don't use challenge infos,
		// todo (jbltx) maybe store them in database ?
		gameServer := GameServer{
			EndpointID:        int64(endpoint.Uint64()),
			IP:                endpoint.IP.String(),
			Port:              int32(endpoint.Port),
			LastHeartbeatDate: time.Now(),
		}
		opts := options.Update().SetUpsert(true) // create a new document if not already here
		update := bson.D{{Key: "$set", Value: gameServer}}
		res, err := gameServersCollection.UpdateOne(context.TODO(), filter, update, opts)
		if err != nil {
			log.Fatal(err)
		}
		if res.UpsertedCount == 0 {
			log.Println("[INFO] CHALLENGE - An endpoint has been updated in the database (" + endpoint.String() + ")")
		} else {
			log.Println("[INFO] CHALLENGE - A new endpoint has been added in the database (" + endpoint.String() + ")")
		}
	}
}

// func setupDatabase(db *mongo.Database) {

// 	collectionNames, colErr := db.ListCollectionNames(context.TODO(), bson.D{})
// 	if colErr != nil {
// 		log.Fatal(colErr)
// 	}

// 	for _, col := range collectionNames {
// 		if dropErr := db.Collection(col).Drop(context.TODO()); dropErr != nil {
// 			log.Fatal(dropErr)
// 		}
// 	}

// 	serversCollectionSchema := bson.M{
// 		"bsonType": "object",
// 		"required": []string{"endpointID", "ip", "port", "lastHeartbeatDate"},
// 		"properties": bson.M{
// 			"endpointID": bson.M{
// 				"bsonType":    "long",
// 				"description": "the endpoint Hash",
// 			},
// 			"ip": bson.M{
// 				"bsonType":    "string",
// 				"description": "the endpoint IP address",
// 			},
// 			"port": bson.M{
// 				"bsonType":    "int",
// 				"maximum":     65535,
// 				"description": "the endpoint Port",
// 			},
// 			"lastHeartbeatDate": bson.M{
// 				"bsonType":    "date",
// 				"description": "the last time when the heartbeat has been received",
// 			},
// 		},
// 	}
// 	serversCollectionValidator := bson.M{
// 		"$jsonSchema": serversCollectionSchema,
// 	}
// 	serversCollectionOpts := options.CreateCollection().SetValidator(serversCollectionValidator)
// 	err := db.CreateCollection(context.TODO(), serversCollectionName, serversCollectionOpts)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	serversCollectionIndices := []mongo.IndexModel{
// 		{
// 			Keys:    bson.D{{"endpointID", 1}},
// 			Options: options.Index().SetUnique(true),
// 		},
// 		{
// 			Keys:    bson.D{{"lastHeartbeatDate", 1}},
// 			Options: options.Index().SetExpireAfterSeconds(serverEntryExpiration),
// 		},
// 	}
// 	opts := options.CreateIndexes().SetMaxTime(2 * time.Second)
// 	db.Collection(serversCollectionName).Indexes().CreateMany(context.TODO(), serversCollectionIndices, opts)

// 	challengesCollectionSchema := bson.M{
// 		"bsonType": "object",
// 		"required": []string{"endpointID", "value"},
// 		"properties": bson.M{
// 			"endpointID": bson.M{
// 				"bsonType":    "long",
// 				"description": "the endpoint Hash",
// 			},
// 			"value": bson.M{
// 				"bsonType":    "int",
// 				"description": "the challenged requested to the gameserver",
// 			},
// 			"updatedAt": bson.M{
// 				"bsonType":    "date",
// 				"description": "last update date",
// 			},
// 		},
// 	}
// 	challengesCollectionValidator := bson.M{
// 		"$jsonSchema": challengesCollectionSchema,
// 	}
// 	challengesCollectionOpts := options.CreateCollection().SetValidator(challengesCollectionValidator)
// 	err = db.CreateCollection(context.TODO(), challengesCollectionName, challengesCollectionOpts)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	challengesCollectionIndices := []mongo.IndexModel{
// 		{
// 			Keys:    bson.D{{"endpointID", 1}},
// 			Options: options.Index().SetUnique(true),
// 		},
// 		{
// 			Keys:    bson.D{{"updatedAt", 1}},
// 			Options: options.Index().SetExpireAfterSeconds(challengeEntryExpiration),
// 		},
// 	}
// 	db.Collection(challengesCollectionName).Indexes().CreateMany(context.TODO(), challengesCollectionIndices, opts)
// }

// func populateDatabase() {

// 	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

// 	gs := GameServer{
// 		EndpointID:        123456789,
// 		IP:                "192.168.2.16",
// 		Port:              27015,
// 		LastHeartbeatDate: time.Now(),
// 	}

// 	res, err := gameServersCollection.InsertOne(ctx, gs)

// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	log.Println(res.InsertedID.(primitive.ObjectID).Hex())

// 	ch := Challenge{
// 		EndpointID: 123456789,
// 		Value:      123,
// 		UpdatedAt:  time.Now(),
// 	}

// 	res, err = challengesCollection.InsertOne(ctx, ch)

// 	log.Println(res.InsertedID.(primitive.ObjectID).Hex())
// }

func (ms *MasterServer) Listen() error {

	mongoClient, err := mongo.NewClient(options.Client().ApplyURI(ms.cfg.Database.URL))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = mongoClient.Connect(ctx)
	defer func() {
		if err = mongoClient.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	db := mongoClient.Database(ms.cfg.Database.Name)

	// // ! Start DEBUG
	// args := os.Args
	// if len(args) > 1 && args[1] == "setup" {
	// 	setupDatabase(db)
	// 	return
	// }
	// // ! End DEBUG

	gameServersCollection = db.Collection(config.ServersCollectionName)
	challengesCollection = db.Collection(config.ChallengesCollectionName)

	// // ! Start DEBUG
	// if len(args) > 1 && args[1] == "populate" {
	// 	populateDatabase()
	// 	return
	// }
	// // ! End DEBUG

	s, err := net.ResolveUDPAddr("udp4", ":"+strconv.Itoa(int(ms.cfg.Port)))
	if err != nil {
		return err
	}

	connection, err := net.ListenUDP("udp4", s)
	if err != nil {
		return err
	}

	defer connection.Close()
	buffer := make([]byte, 1600) // standard MTU size -- no packet should be bigger
	rand.Seed(time.Now().Unix())

	for {
		n, addr, err := connection.ReadFromUDP(buffer)

		if err != nil {
			fmt.Println(err)
			continue
		}

		endpoint := &ServerEndpoint{
			IP:   addr.IP,
			Port: uint16(addr.Port),
		}

		reqHeader := buffer[0]
		var response []byte = nil
		switch reqHeader {
		case valve.RequestServerListHeader:
			response = handleServerListRequest(buffer[1:n-2], endpoint)
		case valve.RequestJoinHeader:
			response = handleJoinRequest(endpoint)
		case valve.RequestQuitHeader:
			if bytes.Equal(buffer, valve.QuitHeader) {
				handleQuitRequest(endpoint)
			}
		case valve.RequestChallengeHeader:
			handleChallengeRequest(buffer[1:n-2], endpoint)
		default:
			continue
		}

		if response != nil {
			_, err = connection.WriteToUDP(response, addr)
		}
	}

	return nil
}
