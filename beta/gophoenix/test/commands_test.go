package test

import (
	"PhoenixOracle/gophoenix/command"
	"flag"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
	"gopkg.in/h2non/gock.v1"
	"io/ioutil"
	"testing"
)

func TestCommandShowJob(t *testing.T) {
	defer CloseGock(t)
	job := NewJob()
	gock.New("http://localhost:8080").
		Get("/jobs/" + job.ID).
		Reply(200).
		JSON(job)

	client := command.Client{ioutil.Discard}

	set := flag.NewFlagSet("test", 0)
	set.Parse([]string{job.ID})
	c := cli.NewContext(nil, set, nil)
	assert.Nil(t, client.ShowJob(c))
}

func TestCommandShowJobNotFound(t *testing.T) {
	defer CloseGock(t)
	gock.New("http://localhost:8080").
		Get("/jobs/bogus-ID").
		Reply(404)

	client := command.Client{ioutil.Discard}

	set := flag.NewFlagSet("test", 0)
	set.Parse([]string{"bogus-ID"})
	c := cli.NewContext(nil, set, nil)
	assert.NotNil(t, client.ShowJob(c))
}

