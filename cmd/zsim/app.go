package main

import (
	"math/rand"
	"net"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/icrowley/fake"
)

type AppLog struct {
	Ts nano.Ts
	//Version int
	Name     string
	App      string
	Location string
	//Location Locator
	ip net.IP
}

type AppModel struct {
	names     []string
	locations map[string][]location
	ip        []net.IP
	apps      []string
	hacker    string
	nhack     int
	moscow    location
}

func NewAppModel() *AppModel {
	a := &AppModel{}
	var locations []location
	for k := 0; k < 20; k++ {
		a.names = append(a.names, fake.FirstName()+" "+fake.LastName())
		ip := net.ParseIP(fake.IPv4())
		locations = append(locations, location{fake.City(), ip})
	}
	a.locations = make(map[string][]location)
	for k := 0; k < 20; k++ {
		name := a.names[k]
		a.locations[name] = []location{locations[rand.Intn(20)]}
		if rand.Intn(2) == 0 {
			a.locations[name] = append(a.locations[name], locations[rand.Intn(20)])
		}
	}
	a.apps = []string{"dropbox", "office365", "salesforce", "youtube", "slack", "amazon", "adp"}
	a.hacker = a.names[0]
	a.moscow.city = "Moscow"
	a.moscow.ip = net.ParseIP(fake.IPv4())
	return a
}

func (a *AppModel) name() string {
	return a.names[rand.Intn(len(a.names))]
}

type location struct {
	city string
	ip   net.IP
}

func (a *AppModel) locationOf(name string) location {
	locations := a.locations[name]
	//pretty.Println(name, locations)
	return locations[rand.Intn(len(locations))]
}

func (a *AppModel) app() string {
	return a.apps[rand.Intn(len(a.apps))]
}

func rare() bool {
	return rand.Intn(20) == 0
}

func (a *AppModel) Next(now nano.Ts) *AppLog {
	ms := int64(rand.Intn(1_000))
	ts := now.Add(ms * 1_000_000)

	app := a.app()
	name := a.name()
	location := a.locationOf(name)
	if a.nhack < 5 && name == a.hacker && rare() {
		a.nhack++
		location.city = "Moscow"
		app = "dropbox"
	}
	return &AppLog{
		Ts:       ts,
		Name:     name,
		App:      app,
		Location: location.city,
	}
}
