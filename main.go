package main

import (
	bt "./twitter"
	"flag"
	"fmt"
	"github.com/ChimeraCoder/anaconda"
	"github.com/cenkalti/rpc2"
	irc "github.com/fluffle/goirc/client"
	cfg "github.com/jebjerg/go-bot/bot/config"
	"net"
	"strings"
)

var api *anaconda.TwitterApi

func init() {
	config = &bt.Twitter_conf{}
	if err := cfg.NewConfig(config, "twitter.json"); err != nil {
		panic(err)
	}

	anaconda.SetConsumerKey(config.ConsumerKey)
	anaconda.SetConsumerSecret(config.ConsumerSecret)
	api = anaconda.NewTwitterApi(config.AccessKey, config.AccessSecret)
}

var config *bt.Twitter_conf
var debug bool

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
			client.Call("privmsg", &bt.PrivMsg{channel, "yes? maybe search"}, nil)
		}
		return nil
	})
	c.Call("register", struct{}{}, nil)

	for _, channel := range config.Channels {
		c.Call("join", channel, nil)
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
					go c.Call("privmsg", &bt.PrivMsg{channel, msg}, nil)
				}
			case anaconda.Tweet:
				msg := fmt.Sprintf("\002\00302@%v\003\002 %v", t.User.ScreenName, bt.FormatTweet(t))
				for _, channel := range config.Channels {
					go c.Call("privmsg", &bt.PrivMsg{channel, msg}, nil)
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
