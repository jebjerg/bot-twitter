package main

import (
	"flag"
	"fmt"
	"github.com/ChimeraCoder/anaconda"
	"github.com/cenkalti/rpc2"
	irc "github.com/fluffle/goirc/client"
	cfg "github.com/jebjerg/go-bot/bot/config"
	"html"
	"net"
	"regexp"
	"strings"
)

type PrivMsg struct {
	Target, Text string
}

var api *anaconda.TwitterApi

func init() {
	config = &twitter_conf{}
	if err := cfg.NewConfig(config, "twitter.json"); err != nil {
		panic(err)
	}

	anaconda.SetConsumerKey(config.ConsumerKey)
	anaconda.SetConsumerSecret(config.ConsumerSecret)
	api = anaconda.NewTwitterApi(config.AccessKey, config.AccessSecret)
}

type twitter_conf struct {
	Channels       []string `json:"channels"`
	BotHost        string   `json:"bot_host"`
	ConsumerKey    string   `json:"consumer_key"`
	ConsumerSecret string   `json:"consumer_secret"`
	AccessKey      string   `json:"access_key"`
	AccessSecret   string   `json:"access_secret"`
}

var config *twitter_conf
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
	flag.BoolVar(&debug, "debug", false, "debug mode")
	flag.Parse()

	conn, err := net.Dial("tcp", config.BotHost)
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
