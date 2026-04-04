//go:build ignore
// +build ignore

// seed_mongodb.go — Mevcut JSON verilerini MongoDB'ye aktarır.
//
// Kullanım (go-backend dizininde):
//   go run seed_mongodb.go
//
// Opsiyonel:
//   MONGODB_URI=mongodb://localhost:27017 go run seed_mongodb.go

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal("Bağlantı hatası:", err)
	}
	defer client.Disconnect(ctx)

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("MongoDB ping başarısız:", err)
	}
	fmt.Println("✓ MongoDB bağlantısı kuruldu:", uri)

	// Her servis kendi izole DB'sini kullanır (PDF gereksinimi)
	authDB     := client.Database("eticaret_auth")
	productDB  := client.Database("eticaret_products")
	addressDB  := client.Database("eticaret_addresses")
	orderDB    := client.Database("eticaret_orders")

	seedCollection(ctx, authDB,    "users",     "auth-service/data/users.json")
	seedCollection(ctx, productDB, "products",  "product-service/data/products.json")
	seedCollection(ctx, addressDB, "addresses", "address-service/data/addresses.json")
	seedCollection(ctx, orderDB,   "orders",    "order-service/data/orders.json")

	// Her DB'nin kendi counters koleksiyonu
	syncCounters(ctx, authDB,    []string{"users"})
	syncCounters(ctx, productDB, []string{"products"})
	syncCounters(ctx, addressDB, []string{"addresses"})
	syncCounters(ctx, orderDB,   []string{"orders"})

	fmt.Println("\n✓ Seed işlemi tamamlandı.")
}

func seedCollection(ctx context.Context, db *mongo.Database, collName, filePath string) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("  ✗ %s: dosya okunamadı (%v), atlanıyor\n", filePath, err)
		return
	}

	var docs []map[string]interface{}
	if err := json.Unmarshal(data, &docs); err != nil {
		fmt.Printf("  ✗ %s: JSON parse hatası (%v)\n", filePath, err)
		return
	}
	if len(docs) == 0 {
		fmt.Printf("  - %s: boş, atlanıyor\n", collName)
		return
	}

	// _id alanını JSON'daki "id" alanından ata
	for i, doc := range docs {
		if id, ok := doc["id"]; ok {
			doc["_id"] = toInt(id)
			delete(doc, "id")
			docs[i] = doc
		}
	}

	coll := db.Collection(collName)
	// Mevcut verileri temizle
	coll.Drop(ctx)

	iDocs := make([]interface{}, len(docs))
	for i, d := range docs {
		iDocs[i] = d
	}

	result, err := coll.InsertMany(ctx, iDocs)
	if err != nil {
		fmt.Printf("  ✗ %s: insert hatası (%v)\n", collName, err)
		return
	}
	fmt.Printf("  ✓ %s: %d kayıt aktarıldı\n", collName, len(result.InsertedIDs))
}

func syncCounters(ctx context.Context, db *mongo.Database, collections []string) {
	counters := db.Collection("counters")

	for _, name := range collections {
		coll := db.Collection(name)
		cursor, err := coll.Find(ctx, bson.M{}, options.Find().SetProjection(bson.M{"_id": 1}))
		if err != nil {
			continue
		}
		var docs []struct {
			ID interface{} `bson:"_id"`
		}
		cursor.All(ctx, &docs)
		cursor.Close(ctx)

		maxID := 0
		for _, d := range docs {
			if id := toInt(d.ID); id > maxID {
				maxID = id
			}
		}

		counters.FindOneAndUpdate(ctx,
			bson.M{"_id": name},
			bson.M{"$set": bson.M{"seq": maxID}},
			options.FindOneAndUpdate().SetUpsert(true),
		)
		fmt.Printf("  ✓ counter[%s] = %d\n", name, maxID)
	}
}

func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return 0
}
