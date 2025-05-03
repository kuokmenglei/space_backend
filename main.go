package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Planet struct {
	ID                int    `json:"id" bson:"id"`
	Name              string `json:"name" bson:"name"`
	CurrentPopulation int    `json:"currentPopulation" bson:"currentPopulation"`
	PictureURL        string `json:"pictureUrl" bson:"pictureUrl"`
}

type Spacecraft struct {
	ID              string `json:"id" bson:"id"`
	Name            string `json:"name" bson:"name"`
	Capacity        int    `json:"capacity" bson:"capacity"`
	Description     string `json:"description" bson:"description"`
	PictureURL      string `json:"pictureUrl" bson:"pictureUrl"`
	CurrentLocation int    `json:"currentLocation" bson:"currentLocation"`
}

var client *mongo.Client

var spacecraftsCollection *mongo.Collection

/*
var (

	planets     = make(map[string]Planet)
	spacecrafts = make(map[string]Spacecraft)
	mutex       = &sync.Mutex{}

)
*/
func connectMongo() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// MongoDB connection URI
	uri := "mongodb://localhost:27017"
	//uri := "mongodb://username:password@localhost:27017"

	// Set client options
	clientOptions := options.Client().ApplyURI(uri)

	var err error
	client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	spacecraftsCollection = client.Database("space").Collection("spacecrafts")

	// Test connection
	// err = client.Ping(ctx, nil)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	log.Println("Connected to MongoDB")
}

func insertMockData() {
	planets := []interface{}{
		Planet{0, "Mercury", 0, "https://upload.wikimedia.org/wikipedia/commons/8/88/Reprocessed_Mariner_10_image_of_Mercury.jpg"},
		Planet{1, "Venus", 0, "https://upload.wikimedia.org/wikipedia/commons/thumb/8/85/Venus_globe.jpg/800px-Venus_globe.jpg"},
		Planet{2, "Earth", 100000, "https://upload.wikimedia.org/wikipedia/commons/thumb/c/cb/The_Blue_Marble_%28remastered%29.jpg/800px-The_Blue_Marble_%28remastered%29.jpg"},
		Planet{3, "Mars", 0, "https://upload.wikimedia.org/wikipedia/commons/thumb/0/02/OSIRIS_Mars_true_color.jpg/800px-OSIRIS_Mars_true_color.jpg"},
		Planet{4, "Jupiter", 0, "https://upload.wikimedia.org/wikipedia/commons/thumb/9/9c/Jupiter%2C_image_taken_by_NASA%27s_Hubble_Space_Telescope%2C_June_2019.png/800px-Jupiter%2C_image_taken_by_NASA%27s_Hubble_Space_Telescope%2C_June_2019.png"},
		Planet{5, "Saturn", 0, "https://upload.wikimedia.org/wikipedia/commons/thumb/e/ea/8423_20181_1saturn2016.jpg/1920px-8423_20181_1saturn2016.jpg"},
		Planet{6, "Uranus", 0, "https://upload.wikimedia.org/wikipedia/commons/thumb/c/c9/Uranus_as_seen_by_NASA%27s_Voyager_2_%28remastered%29_-_JPEG_converted.jpg/800px-Uranus_as_seen_by_NASA%27s_Voyager_2_%28remastered%29_-_JPEG_converted.jpg"},
		Planet{7, "Neptune", 0, "https://upload.wikimedia.org/wikipedia/commons/0/06/Neptune.jpg"},
	}

	spacecraft := Spacecraft{
		ID:              "prispax",
		Name:            "Prispax",
		Capacity:        10000,
		Description:     "Astrolux Odyssey: luxury meets space travel...",
		PictureURL:      "",
		CurrentLocation: 2,
	}

	ctx := context.Background()
	db := client.Database("space")
	db.Collection("planets").Drop(ctx)
	db.Collection("spacecrafts").Drop(ctx)

	_, _ = db.Collection("planets").InsertMany(ctx, planets)
	_, _ = db.Collection("spacecrafts").InsertOne(ctx, spacecraft)
	log.Println("Mock data inserted")
}

func getPlanets(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	cursor, err := client.Database("space").Collection("planets").Find(ctx, bson.M{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var results []Planet
	if err := cursor.All(ctx, &results); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(results)
}

func getSpacecrafts(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	cursor, err := client.Database("space").Collection("spacecrafts").Find(ctx, bson.M{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var results []Spacecraft
	if err := cursor.All(ctx, &results); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(results)
}

func getSpacecraftById(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var sc Spacecraft
	err := spacecraftsCollection.FindOne(context.TODO(), bson.M{"id": id}).Decode(&sc)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	json.NewEncoder(w).Encode(sc)
}

func buildSpacecraft(w http.ResponseWriter, r *http.Request) {
	var sc Spacecraft
	if err := json.NewDecoder(r.Body).Decode(&sc); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	_, err := spacecraftsCollection.InsertOne(context.TODO(), sc)
	if err != nil {
		http.Error(w, "Failed to insert spacecraft", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sc)
}

func destroySpacecraftById(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	_, err := spacecraftsCollection.DeleteOne(context.TODO(), bson.M{"id": id})
	if err != nil {
		http.Error(w, "Failed to delete spacecraft", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func sendSpacecraftToPlanet(w http.ResponseWriter, r *http.Request) {
	var mission struct {
		SpacecraftID   string `json:"spacecraftId"`
		TargetPlanetID int    `json:"targetPlanetId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&mission); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	update := bson.M{"$set": bson.M{"currentLocation": mission.TargetPlanetID}}
	_, err := spacecraftsCollection.UpdateOne(context.TODO(), bson.M{"id": mission.SpacecraftID}, update)
	if err != nil {
		http.Error(w, "Failed to update spacecraft location", http.StatusInternalServerError)
		return
	}

	resp := map[string]string{
		"message": "Spacecraft " + mission.SpacecraftID + " sent to planet " + string(rune(mission.TargetPlanetID)),
	}
	json.NewEncoder(w).Encode(resp)
}

func enableCors(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		h.ServeHTTP(w, r)
	})
}

func main() {
	connectMongo()
	//insertMockData()

	r := mux.NewRouter()
	r.HandleFunc("/planets", getPlanets).Methods("GET")
	r.HandleFunc("/spacecrafts", getSpacecrafts).Methods("GET")
	r.HandleFunc("/spacecrafts/{id}", getSpacecraftById).Methods("GET")
	r.HandleFunc("/spacecrafts", buildSpacecraft).Methods("POST")
	r.HandleFunc("/spacecrafts/{id}", destroySpacecraftById).Methods("DELETE")
	r.HandleFunc("/missions/send", sendSpacecraftToPlanet).Methods("POST")

	handlerWithCORS := enableCors(r)
	log.Println("Server running at :8080")
	log.Fatal(http.ListenAndServe(":8080", handlerWithCORS))
}
