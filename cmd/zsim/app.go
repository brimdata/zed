package main

import (
	"math/rand"
	"net"

	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/icrowley/fake"
)

type AppLog struct {
	Name     string `zng:"name"`
	App      string `zng:"app"`
	Location string `zng:"location"`
}

type AppLogv3 struct {
	Name     string `zng:"name"`
	App      string `zng:"app"`
	Location string `zng:"location"`
	IP       net.IP `zng:"ip"`
}

type Location struct {
	City string `zng:"city"`
	IP   net.IP `zng:"ip"`
}

type AppLogv2 struct {
	Name     string   `zng:"name"`
	App      string   `zng:"app"`
	Location Location `zng:"location"`
}

type AppModel struct {
	version   int
	names     []string
	locations map[string][]Location
	ip        []net.IP
	apps      []string
	hacker    string
	nhack     int
	moscow    Location
}

func NewAppModel(version int) *AppModel {
	a := &AppModel{version: version}
	var locations []Location
	for k := 0; k < 20; k++ {
		a.names = append(a.names, fake.FirstName()+" "+fake.LastName())
		ip := net.ParseIP(fake.IPv4())
		locations = append(locations, Location{fake.City(), ip})
	}
	a.locations = make(map[string][]Location)
	for k := 0; k < 20; k++ {
		name := a.names[k]
		a.locations[name] = []Location{locations[rand.Intn(20)]}
		if rand.Intn(2) == 0 {
			a.locations[name] = append(a.locations[name], locations[rand.Intn(20)])
		}
	}
	a.apps = []string{"dropbox", "office365", "salesforce", "youtube", "slack", "amazon", "adp"}
	a.hacker = a.names[0]
	a.moscow.City = "Moscow"
	a.moscow.IP = net.ParseIP(fake.IPv4())
	return a
}

func (a *AppModel) name() string {
	return a.names[rand.Intn(len(a.names))]
}

func (a *AppModel) locationOf(name string) Location {
	locations := a.locations[name]
	return locations[rand.Intn(len(locations))]
}

func (a *AppModel) app() string {
	return a.apps[rand.Intn(len(a.apps))]
}

func rare() bool {
	return rand.Intn(20) == 0
}

func (a *AppModel) Next(zctx *resolver.Context) *zng.Record {
	app := a.app()
	name := a.name()
	location := a.locationOf(name)
	if a.nhack < 5 && name == a.hacker && rare() {
		a.nhack++
		location = a.moscow
		app = "dropbox"
	}
	var rec *zng.Record
	switch a.version {
	default:
		rec, _ = resolver.MarshalRecord(zctx,
			&AppLog{
				Name:     name,
				App:      app,
				Location: location.City,
			})
	case 2:
		rec, _ = resolver.MarshalRecord(zctx,
			&AppLogv2{
				Name:     name,
				App:      app,
				Location: location,
			})

	case 3:
		rec, _ = resolver.MarshalRecord(zctx,
			&AppLogv3{
				Name:     name,
				App:      app,
				Location: location.City,
				IP:       location.IP,
			})
	}
	return rec
}
