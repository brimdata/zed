package main

import (
	"math/rand"
	"net"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zng/resolver"
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
	zctx   *resolver.Context
	names  []string
	cities []string
	ip     []net.IP
	apps   []string
	hacker string
}

func NewAppModel(zctx *resolver.Context) *AppModel {
	a := &AppModel{}
	for k := 0; k < 20; k++ {
		a.names = append(a.names, fake.FullName())
		a.cities = append(a.cities, fake.City())
	}
	a.apps = []string{"dropbox", "office365", "salesforce", "youtube", "slack", "amazon", "adp"}
	a.hacker = a.names[0]
	return a
}

func (a *AppModel) name() string {
	return a.names[rand.Intn(len(a.names))]
}

type location struct {
	city string
	ip   net.IP
}

func (a *AppModel) city() string {
	return a.cities[rand.Intn(len(a.cities))]
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

	city := a.city()
	name := a.name()
	if name == a.hacker && rare() {
		city = "Moscow"
	}
	return &AppLog{
		Ts:       ts,
		Name:     name,
		App:      a.app(),
		Location: city,
	}
}
