package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ChimeraCoder/anaconda"
	"github.com/cenkalti/rpc2"
	irc "github.com/fluffle/goirc/client"
	"html"
	"io/ioutil"
	"net"
	"regexp"
	"strings"
)

type PrivMsg struct {
	Target, Text string
}

var api *anaconda.TwitterApi

func init() {
	var err error
	config, err = NewConfig("./twitter.json")
	if err != nil {
		panic(err)
	}

	anaconda.SetConsumerKey(config.ConsumerKey)
	anaconda.SetConsumerSecret(config.ConsumerSecret)
	api = anaconda.NewTwitterApi(config.AccessKey, config.AccessSecret)
}

type Config struct {
	Channels       []string `json:"channels"`
	ConsumerKey    string   `json:"consumer_key"`
	ConsumerSecret string   `json:"consumer_secret"`
	AccessKey      string   `json:"access_key"`
	AccessSecret   string   `json:"access_secret"`
}

func (c *Config) Save(path string) error {
	if data, err := json.MarshalIndent(c, "", "    "); err != nil {
		return err
	} else {
		return ioutil.WriteFile(path, data, 600)
	}
}

func NewConfig(path string) (*Config, error) {
	config := &Config{}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, config)
	return config, err
}

func Remove(element string, elements *[]string) error {
	index := -1
	for i, e := range *elements {
		if e == element {
			index = i
		}
	}
	if index != -1 {
		*elements = append((*elements)[:index], (*elements)[index+1:]...)
	} else {
		return fmt.Errorf("element (%v) not found in (%v)", element, elements)
	}
	return nil
}

var config *Config
var debug bool

func FormatTweet(tweet anaconda.Tweet) string {
	prefix := ""
	if tweet.Retweeted {
		tweet = *tweet.RetweetedStatus
		prefix = fmt.Sprintf("RT: @%v", tweet.User.ScreenName)
	}
	output := tweet.Text
	for i := len(tweet.Entities.Urls) - 1; i >= 0; i-- {
		url := tweet.Entities.Urls[i]
		output = output[:url.Indices[0]] + url.Expanded_url + output[url.Indices[1]:]
	}
	output = prefix + output
	output = html.UnescapeString(output)

	users := regexp.MustCompile(`(@(\w){1,15})`)
	hashtag := regexp.MustCompile(`(#\w+)`)

	output = users.ReplaceAllString(output, "\002\00302$1\003\002")
	output = hashtag.ReplaceAllString(output, "\00304$1\003")
	return output
}

func main() {
	flag.BoolVar(&debug, "debug", false, "debug mode (localhost xml feed)")
	flag.Parse()

	conn, err := net.Dial("tcp", "localhost:1234")
	if err != nil {
		panic(err)
	}
	c := rpc2.NewClient(conn)
	go c.Run()

	// just for kicks
	c.Handle("privmsg", func(client *rpc2.Client, args *irc.Line, reply *bool) error {
		channel, line := args.Args[0], args.Args[1]
		if strings.Fields(line)[0] == ".twitter" {
			client.Call("privmsg", &PrivMsg{channel, "yes? maybe search"}, &reply)
		}
		return nil
	})
	var reply bool
	c.Call("register", struct{}{}, &reply)

	for _, channel := range config.Channels {
		c.Call("join", channel, &reply)
	}

	go func() {
		if api == nil {
			return
		}
		stream := api.UserStream(nil)
		dead := false
		for {
			t := <-stream.C
			switch t := t.(type) {
			default:
				fmt.Printf("Unexpected type %T\n", t)
			case anaconda.FriendsList:
				msg := fmt.Sprintf("initialized twitter following \002%d\002 accounts", len(t))
				for _, channel := range config.Channels {
					go c.Call("privmsg", &PrivMsg{channel, msg}, &reply)
				}
			case anaconda.Tweet:
				msg := fmt.Sprintf("\002\00302@%v\003\002 %v", t.User.ScreenName, FormatTweet(t))
				for _, channel := range config.Channels {
					go c.Call("privmsg", &PrivMsg{channel, msg}, &reply)
				}
			case anaconda.DisconnectMessage:
				fmt.Println("oh no, gotta go", t)
				dead = true
				break
			}
			if dead {
				break
			}
		}
	}()
	forever := make(chan bool)
	<-forever
}
