// Embedded in this article https://medium.com/p/c98e491015b6
// https://asten.io/go-graphql-getting-started-c98e491015b6
package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/rs/cors"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//////// MAIN ////////
func main() {
	// Create a handler for /graphql which passes cors for remote requests
	http.Handle("/graphql", cors.Default().Handler(&relay.Handler{Schema: graphqlSchema}))

	// Write a GraphiQL page to /
	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(page)
	}))

	// ListenAndServe starts an HTTP server with a given address and handler.
	log.Fatal(http.ListenAndServe(":8080", nil))
}

//////// MONGODB ////////
const host string = "localhost"

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

// GetMongo returns the session and a reference to the post collection.
func GetMongo(col string) (*mgo.Session, *mgo.Collection) {
	maxWait := time.Duration(5 * time.Second)
	session, err := mgo.DialWithTimeout(host, maxWait)

	if err != nil {
		log.Fatal(err)
	}

	collection := session.DB("minimalGraphql").C(col)

	return session, collection
}

//////// GRAPHQL ////////
var graphqlSchema *graphql.Schema

// Schema describes the data that we ask for
var Schema = `
    schema {
        query: Query
    }
    # The Query type represents all of the entry points.
    type Query {
        post(slug: String!): Post
    }
    type Post {
        id: ID!
        slug: String!
        title: String!
    }
    `

//////// INIT ////////
func init() {

	// MustParseSchema parses a GraphQL schema and attaches the given root resolver.
	// It returns an error if the Go type signature of the resolvers does not match the schema.
	graphqlSchema = graphql.MustParseSchema(Schema, &Resolver{})

	log.Println("Seeding mock data to MongoDB")
	// Call GetMongo, session and reference to the post collection
	session, collection := GetMongo("post")
	// Close the session so its resources may be put back in the pool or collected, depending on the case.
	defer session.Close()

	// Cleanup finds all documents matching the provided selector document
	// and removes them from the database. So we make sure the db is empty before inserting mock data.
	Cleanup("post")

	// The mock data that we insert.
	err := collection.Insert(
		bson.M{"ID": "1", "title": "First post", "slug": "first-post"},
		bson.M{"ID": "2", "title": "Second post", "slug": "second-post"},
		bson.M{"ID": "3", "title": "Third post", "slug": "third-post"},
	)

	if err != nil {
		log.Fatal(err)
	}
	log.Println("Mock data added successfully!")
}

//////// RESOLVER ////////
// In order to respond to queries, a schema needs to have resolve functions for all fields.
// Go’s structs are typed collections of fields. They’re useful for grouping data together to form records.
type Resolver struct{}

type post struct {
	ID    graphql.ID
	Slug  string
	Title string
}

type postResolver struct {
	s *post
}

type searchResultResolver struct {
	result interface{}
}

// Slices can be created with the built-in make function; this is how we create dynamically-sized arrays.
var postData = make(map[string]*post)

// Post resolves the Post queries.
func (r *Resolver) Post(args struct{ Slug string }) *postResolver {

	// One result is a pointer to type post.
	oneResult := &post{}

	// Call GetMongo, session and reference to the post collection
	session, collection := GetMongo("post")
	// Close the session so its resources may be put back in the pool or collected, depending on the case.
	defer session.Close()

	// Inside the collection, find by slug and return all fields.
	err := collection.Find(bson.M{"slug": args.Slug}).Select(bson.M{}).One(&oneResult)
	if err != nil {
		fmt.Println(err)
	}

	// Make a type postResolver out of oneResult.
	if s := oneResult; s != nil {
		return &postResolver{oneResult}
	}
	return nil
}

// Resolve each field to respond to queries.
func (r *postResolver) ID() graphql.ID {
	return r.s.ID
}

func (r *postResolver) Slug() string {
	return r.s.Slug
}

func (r *postResolver) Title() string {
	return r.s.Title
}

//////// GRAPHiQL ////////
var page = []byte(`
    <!DOCTYPE html>
    <html>
        <head>
            <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/graphiql/0.10.2/graphiql.css" />
            <script src="https://cdnjs.cloudflare.com/ajax/libs/fetch/1.1.0/fetch.min.js"></script>
            <script src="https://cdnjs.cloudflare.com/ajax/libs/react/15.5.4/react.min.js"></script>
            <script src="https://cdnjs.cloudflare.com/ajax/libs/react/15.5.4/react-dom.min.js"></script>
            <script src="https://cdnjs.cloudflare.com/ajax/libs/graphiql/0.10.2/graphiql.js"></script>
        </head>
        <body style="width: 100%; height: 100%; margin: 0; overflow: hidden;">
            <div id="graphiql" style="height: 100vh;">Loading...</div>
            <script>
                function graphQLFetcher(graphQLParams) {
                    return fetch("/graphql", {
                        method: "post",
                        body: JSON.stringify(graphQLParams),
                        credentials: "include",
                    }).then(function (response) {
                        return response.text();
                    }).then(function (responseBody) {
                        try {
                            return JSON.parse(responseBody);
                        } catch (error) {
                            return responseBody;
                        }
                    });
                }
                ReactDOM.render(
                    React.createElement(GraphiQL, {fetcher: graphQLFetcher}),
                    document.getElementById("graphiql")
                );
            </script>
        </body>
    </html>
    `)