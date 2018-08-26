# routepiler

This is an HTTP router compiler, it was sort of a joke that I took way, way.. way too far. Quick summary of project layout:

 - routepiler: Package routepiler provides route compilation, meant to be called from within unit tests to ensure routes stay up to date.
 - cmd/routepiler: Package main implements the routepiler command line interface.
 - internal/token: Package token provides constants for lexical classification of patterns through lexemes which map one or more characters within tokens to a source position.
 - internal/scanner: Package scanner converts one or more route inputs into tokens.
 - internal/parser: Package parser verifies a token stream is correct before generating one or more route objects ready for analysis.
 - internal/analyze: Package analyze runs the validation & scoring heuristics of each route compiler to select the best code generation method for that route.
 - internal/compile: Package compile generates code from analyzed routes using the currently configured backend.
 - internal/backend: Package backend defines the common interface which all backends must implement.
 - internal/backend/gosrc: Package gosrc implements the backend interface by generating Go source code from one or more analyzed routes.
 - internal/backend/pysrc: Package pysrc implements the backend interface by generating Python source code from one or more analyzed routes.


Note: This project is a couple years old now and I don't see myself ever
finishing it, but I plan on cleaning up and committing all the other packages at
some point. It was originally named "handy", short for "http handler"- I didn't
really think that through. Regardless that is why the original play I put up at
https://play.golang.org/p/yJmz6qPx_N was named "handy" and you may find random
references to "handy" throughout the code. The snippet below shows an out of
date usage example, the final API made it convenient to generate & test your
code within unit tests.


```Go
//go:generate -command handy -o ./main.handy.go ./main.go
package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

// So I started looking at routers again last night for the first time in a
// couple years and kinda laughed at all the webscale going on and how everyone
// was "the fastest in the world".. some do things like delay getting parameters
// from the route until requested to benchmark nicer. Despite mocking this
// pursuit in the past.. I wrote another one.
//
// Reason is the ones I liked the API for have a full "framework" or stack of
// middleware, some pull in tons of dependencies. I thought it would be nice to
// have a strongly typed router, meaning you could access your Params as the
// T of the type they are.
//
// With runtime router generation you are limited to a small set of generic
// algorithms that have to work well from an unknown set. With code generation
// you know all the paths up front and can use heuristics to select a best path.
//
// Right now I have only two naive generators, the first being based on
// something similar to[1] (lookup table) which is selected for 100% static
// routes with no params this can give route selection as cheap as a single
// memory access from a fixed size stack allocated array which elides bounds
// checks. I think it's as web scale as you can go at a couple nanoseconds.
//
// The second one walks each path segment, allocating the parameters into the
// associated types fields. This makes zero allocation routing possible as we
// satisfy escape analysis. Currently the implementation is ugly and ignores
// edge cases as well as spitting out errors for ambigious routes but it could
// be easily improved.
//
// [1] https://github.com/cstockton/exp/blob/master/archive/lut/lookup.go

// Here is an example for requesting a handler that does nothing but access each
// param, verifying it's value. It's probably going to get about 0.5x-2x slower
// as some of the cases it doesn't handle are closed up like a flag for unicode
// since right now it scans the string as a byte sequence.
//
// Url is:
//
//   /orgs/:org/teams/:team/users/:user
//
// In httprouter:
//
// func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
// 	if ps.ByName(`user`) != `cstockton` {
// 		panic(`fail`)
// 	}
// 	if ps.ByName(`team`) != `acmeteam` {
// 		panic(`fail`)
// 	}
// 	if ps.ByName(`org`) != `acmeorg` {
// 		panic(`fail`)
// 	}
// }
//
// In handy:
//
// func (u *Users) ServeHTTP(w http.ResponseWriter, r *http.Request) {
// 	if u.User != `cstockton` {
// 		panic(`fail`)
// 	}
// 	if u.Team != `acmeteam` {
// 		panic(`fail`)
// 	}
// 	if u.Org != `acmeorg` {
// 		panic(`fail`)
// 	}
// }
//
//
// BenchmarkRoutes/HttpRouter-24         	 3000000	       530 ns/op	      96 B/op	       1 allocs/op
// BenchmarkRoutes/Handy-24              	10000000	       109 ns/op	       0 B/op	       0 allocs/op
// PASS

func main() {

	// Initialize your router however you want.
	r := &Router{app: &App{}}

	// We need a Root, Echo and Create func, if nil the router will panic.
	// @TODO Add a Err() method in gen code for a runtime check.
	r.Root, r.Echo = http.NotFoundHandler(), Echo

	// The code generated in the main.handy.go file defines a ServeHTTP method for
	// each struct with appropriate path tags.
	log.Fatal(http.ListenAndServe(":8080", r))
}

type App struct {
	DBConn       bool  // Could be a real db conn.
	OtherService *bool // Reference to another service.
}

// Router is a struct containing the routes. If it's a function type it is used
// to serve the request. If it's a named type it will be expected to have a
// ServeHTTP method or a matching HTTP method in its method set.
//
// @TODO I'm still determining the best way to define the routes.
type Router struct {

	// The name may match against anything that satisfies the http.Handler
	// interface.
	Root http.Handler `get:"/"`

	// In addition it may be a func, or a func that returns an error.
	//
	// @TODO The error currently doesn't propagate anywhere, but does allow
	// chaining your own handlers, it just is ignored at the top of the route.
	Date func(http.ResponseWriter, *http.Request)       `path:"/date"`
	Echo func(http.ResponseWriter, *http.Request) error `get:"/echo"`

	// When not set it will attempt to find a matching signature with the same
	// field name, or the field name prefixed with `handle`. Here we specify a
	// method explicitly, though better written as get:"/v1/time"
	Time http.Handler `path:"/time" method:"get" func:"handleTime"`

	// If the value is a named type, as long as it satisfies the requirements
	// above it will be initialized to the zero value and used for the request.
	Orgs Orgs `get:"/orgs"`

	// The same struct can be used for multiple routes, but it must have a field
	// to accomodate each parameter. Here we specify to use the GetUser method of
	// the Orgs struct.
	Org Orgs `get:"/orgs/:org" func:"GetOrg"`

	// Users is a child of Orgs, here we only define a path without a method. So
	// the Users struct should be a http.Handler OR have a method matching the
	// incoming request type. i.e.: Users.Get(...)
	Users Users `path:"/orgs/:org/users"`

	// Route requests for a user to GetUser method within the Users methodset.
	User Users `get:"/orgs/:org/users/:user" func:"GetUser"`

	// Allow notifying a User
	Notify Users `get:"/orgs/:org/users/:user/notify/:when" func:"Notify"`

	// We will setup the router with some shared state. This could be a pointer to
	// your shared application struct and passed to your handlers by specifying
	// a extra parameter of type *Router for a full reference, or to a T that does
	// not have route tags.
	app *App

	// Create a Org, requires a reference to the *App state. We could assign a
	// Func when we create our router, i.e.:
	//
	//  Create func(http.ResponseWriter, *http.Request, *App) error `post:"/orgs"`
	//
	// Instead we just add the *App parameter to the Orgs Post method to hint at
	// the code generation to see it has a *App Type in the params. The code gen
	// simply passes it's "r.app" to the call.
	Create Orgs `post:"/orgs"`
}

func Time(w http.ResponseWriter, r *http.Request) error {
	fmt.Printf("[Time] Get: %v at %v\n", r.URL.Path, time.Now())
	return nil
}

var handleTime = Time

func Echo(w http.ResponseWriter, r *http.Request) error {
	b, err := ioutil.ReadAll(io.LimitReader(r.Body, 8))
	if err != nil {
		return err
	}
	fmt.Printf("[Echo] Get: %v echo %q\n", r.URL.Path, string(b))
	return nil
}

// Orgs contains a collection of User.
type Orgs struct {

	// Org is part of the Users route, the router will assign this value.
	Org string
}

func (h *Orgs) Get(w http.ResponseWriter, r *http.Request, app *App) {
	fmt.Printf("[Users] Get: %v :org(%v) app(%v)\n", r.URL.Path, h.Org, app)
}

func (h *Orgs) Post(w http.ResponseWriter, r *http.Request, app *App) {
	fmt.Printf("[Orgs] Post: %v :org(%v) app(%v)\n", r.URL.Path, h.Org, app)
}

// Users contains a collection of User.
type Users struct {
	// User is a child of the Orgs route, embedding *Orgs means we can access the
	// Org params without repeating the arguments here. We could have also just
	// defined an `Org string` field.
	// @TODO Currently defining a Org field is not possible, since the codegen
	// makes a single linear pass through the Path it assigns the parrent params
	// before entering the child route.
	*Orgs

	// User is the :user parameter, the min and max tags define path length
	// boundaries for this route parameter.
	User string `min:"3" max:"20"`

	// When is used for an example of a non-string param type.
	When time.Duration

	// User information, not part of the path.
	Since time.Time `min:"01-01-1850" max:"now"`
	ID    string
	First string
	Last  string
	Age   int `min:"18" max:"120"`
}

func (h *Users) Get(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[Users] Get: %v :org(%v)\n", r.URL.Path, h.Org)
}

func (h *Users) Post(w http.ResponseWriter, r *http.Request) error {
	fmt.Printf("[Users] Get: %v :org(%v)\n", r.URL.Path, h.Org)
	return nil
}

func (h *Users) GetUser(w http.ResponseWriter, r *http.Request) error {
	fmt.Printf("[GetUser] Get: %v :org(%v) :user(%v)\n",
		r.URL.Path, h.Org, h.User)
	return nil
}

func (h *Users) Notify(w http.ResponseWriter, r *http.Request) error {
	fmt.Printf("[Notify] Get: %v :org(%v) :user(%v) when(%v)\n",
		r.URL.Path, h.Org, h.User, h.When)
	return nil
}
```
