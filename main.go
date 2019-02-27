package main

import (
	"encoding/json"
	"fmt"
	"log"
	"github.com/graphql-go/graphql"
	"io/ioutil"
	"net/http"
	"time"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var tutorials []Tutorial
//////// MONGODB ////////
const host string = "localhost"

func GetMongo(col string) (*mgo.Session, *mgo.Collection) {
	maxWait := time.Duration(5 * time.Second)
	session, err := mgo.DialWithTimeout(host, maxWait)

	if err != nil {
		log.Fatal(err)
	}

	collection := session.DB("miniGraphql").C(col)

	return session, collection
}

// Cleanup will remove all mock data from the database.
func Cleanup(col string) {
	log.Println("Cleaning up MongoDB...")
	session, collection := GetMongo(col)
	defer session.Close()

	_, err := collection.RemoveAll(bson.M{})
	if err != nil {
		log.Fatal(err)
	}
}

////// INIT ////////
func init() {
	log.Println("Seeding mock data to MongoDB")
	// Call GetMongo, session and reference to the post collection
	session, collection := GetMongo("tutorial")
	// Close the session so its resources may be put back in the pool or collected, depending on the case.
	defer session.Close()

	// Cleanup finds all documents matching the provided selector document
	// and removes them from the database. So we make sure the db is empty before inserting mock data.
	Cleanup("tutorial")

	// The mock data that we insert.
	// err := collection.Insert(
	// 	bson.M{"id": 1, "title": "Go GraphQL Tutorial", "author": bson.M{"Name": "dat", "Tutorials": 1}, "comment": bson.M{"body": "First Comment"}},
	// 	bson.M{"id": 2, "title": "Second tut"},
	// 	bson.M{"id": 3, "title": "Third tut"},
	// )

	author := &Author{Name: "Elliot Forbes", Tutorials: []int{1, 2}}
	tutorial := Tutorial{
		ID:     1,
		Title:  "Go GraphQL Tutorial",
		Author: *author,
		Comments: []Comment{
			Comment{Body: "First Comment"},
		},
	}
	tutorial2 := Tutorial{
		ID:     2,
		Title:  "Go GraphQL Tutorial - Part 2",
		Author: *author,
		Comments: []Comment{
			Comment{Body: "Second Comment"},
		},
	}

	// tutorials = append(tutorials, tutorial)
	// tutorials = append(tutorials, tutorial2)

	err := collection.Insert(tutorial, tutorial2)

	if err != nil {
		log.Fatal(err)
	}
	err = collection.Find(nil).All(&tutorials)
	if err != nil {
		fmt.Println(err)
	}

	log.Println("Mock data added successfully!")
}

type Tutorial struct {
	ID       int
	Title    string
	Author   Author
	Comments []Comment
}

type Author struct {
	Name      string
	Tutorials []int
}

type Comment struct {
	Body string
}

// var tutorials = populate()
func populate() []Tutorial {
	author := &Author{Name: "Elliot Forbes", Tutorials: []int{1, 2}}
	tutorial := Tutorial{
		ID:     1,
		Title:  "Go GraphQL Tutorial",
		Author: *author,
		Comments: []Comment{
			Comment{Body: "First Comment"},
		},
	}
	tutorial2 := Tutorial{
		ID:     2,
		Title:  "Go GraphQL Tutorial - Part 2",
		Author: *author,
		Comments: []Comment{
			Comment{Body: "Second Comment"},
		},
	}

	var tutorials []Tutorial
	tutorials = append(tutorials, tutorial)
	tutorials = append(tutorials, tutorial2)

	session, collection := GetMongo("tutorial")
	defer session.Close()
	err := collection.Insert(tutorial, tutorial2)
	if err != nil {
		log.Fatal(err)
	}

	return tutorials
}

var authorType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Author",
		Fields: graphql.Fields{
			"Name": &graphql.Field{
				Type: graphql.String,
			},
			"Tutorials": &graphql.Field{
				Type: graphql.NewList(graphql.Int),
			},
		},
	},
)

var commentType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Comment",
		Fields: graphql.Fields{
			"body": &graphql.Field{
				Type: graphql.String,
			},
		},
	},
)

var tutorialType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Tutorial",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.Int,
			},
			"title": &graphql.Field{
				Type: graphql.String,
			},
			"author": &graphql.Field{
				Type: authorType,
			},
			"comments": &graphql.Field{
				Type: graphql.NewList(commentType),
			},
		},
	},
)

var mutationType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Mutation",
	Fields: graphql.Fields{
		"create": &graphql.Field{
			Type:        tutorialType,
			Description: "Create a new Tutorial",
			Args: graphql.FieldConfigArgument{
				"title": &graphql.ArgumentConfig{
					Type: graphql.NewNonNull(graphql.String),
				},
			},
			Resolve: func(params graphql.ResolveParams) (interface{}, error) {
				tutorial := Tutorial{
					Title: params.Args["title"].(string),
				}
				tutorials = append(tutorials, tutorial)
				return tutorial, nil
			},
		},
	},
})

func main() {
	// Schema
	fields := graphql.Fields{
		"tutorial": &graphql.Field{
			Type:        tutorialType,
			Description: "Get Tutorial By ID",
			Args: graphql.FieldConfigArgument{
				"id": &graphql.ArgumentConfig{
					Type: graphql.Int,
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				id, ok := p.Args["id"].(int)
				if ok {
					// Find tutorial
					for _, tutorial := range tutorials {
						if int(tutorial.ID) == id {
							return tutorial, nil
						}
					}
				}
				return nil, nil
			},
		},
		"list": &graphql.Field{
			Type:        graphql.NewList(tutorialType),
			Description: "Get Tutorial List",
			Resolve: func(params graphql.ResolveParams) (interface{}, error) {
				// // Call GetMongo, session and reference to the post collection
				// session, collection := GetMongo("tutorial")
				// // Close the session so its resources may be put back in the pool or collected, depending on the case.
				// defer session.Close()
				// // Inside the collection, find by slug and return all fields.
				// err := collection.Find(nil).All(&tutorials)
				// if err != nil {
				// 	fmt.Println(err)
				// }
				// // log.Println(tutorials)
				return tutorials, nil
			},
		},
	}
	rootQuery := graphql.ObjectConfig{Name: "RootQuery", Fields: fields}
	schemaConfig := graphql.SchemaConfig{
		Query: graphql.NewObject(rootQuery),
		Mutation: mutationType,
	}
	schema, err := graphql.NewSchema(schemaConfig)
	if err != nil {
		log.Fatalf("failed to create new schema, error: %v", err)
	}

	// Query
	// query := `
	// 		mutation {
	// 			create(title: "Hello World") {
	// 					title
	// 			}
	// 		}
	// `
	query := `
	{
		list {
			id
			title
			comments {
				body
			}
			author {
					Name
					Tutorials
			}
		}
	}
	`
	params := graphql.Params{Schema: schema, RequestString: query}
	r := graphql.Do(params)
	if len(r.Errors) > 0 {
		log.Fatalf("failed to execute graphql operation, errors: %+v", r.Errors)
	}
	rJSON, _ := json.Marshal(r)
	fmt.Printf("%s \n", rJSON)

	// Query
	// query = `
	// 	{
	// 		list {
	// 			id
	// 			title
	// 		}
	// 	}
	// `
	// params = graphql.Params{Schema: schema, RequestString: query}
	// r = graphql.Do(params)
	// if len(r.Errors) > 0 {
	// 	log.Fatalf("failed to execute graphql operation, errors: %+v", r.Errors)
	// }
	// rJSON, _ = json.Marshal(r)
	// fmt.Printf("%s \n", rJSON)

	// log.Println(tutorials)

	http.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		// for post request
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal(err)
		}
		result := executeQuery(string(reqBody), schema)

		// for get request
		// result := executeQuery(r.URL.Query().Get("query"), schema)

		//for local query
		// result := executeQuery(query, schema)
		
		json.NewEncoder(w).Encode(result)
	})

	fmt.Println("Now server is running on port 8080")
	// fmt.Println("Test with Get      : curl -g 'http://localhost:8080/graphql?query={user(id:\"1\"){name}}'")
	http.ListenAndServe(":8080", nil)
}

func executeQuery(query string, schema graphql.Schema) *graphql.Result {
	result := graphql.Do(graphql.Params{
		Schema:        schema,
		RequestString: query,
	})
	if len(result.Errors) > 0 {
		fmt.Printf("wrong result, unexpected errors: %v", result.Errors)
	}
	return result
}